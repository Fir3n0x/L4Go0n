package cmd

import (
	"fmt"
	"net"
	"strings"

	wsmanager "github.com/Fir3n0x/my-c2-dashboard/ws"
)

// Handle utility functions

// Retrieve host and port value from conn.RemoteAddr() function <host>:<port>
func extractHostAndPort(addr string) (string, string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Printf("[!] SplitHostPort failed for addr %s: %v", addr, err)
		return "—", "—"
	}

	return host, port
}

// Function to send commands to a client
func SendCommand(id string, cmd string) error {
	MyCommandStore.Load()
	ClientsMu.Lock()
	client, exists := Clients[id]
	ClientsMu.Unlock()

	// Check if the client really exists
	if !exists {
		err := fmt.Errorf("client %s not found", id)
		LogInfo.Printf("[!] %v", err)
		return err
	}

	if client.Reachable && client.Conn != nil {
		// Client already connected
		_, err := client.Conn.Write([]byte(cmd + "\n"))
		if err != nil {
			LogInfo.Printf("[-] Failed to send queued command to %s: %v", id, err)
			return err
		}
		LogInfo.Printf("[+] Sent queued command \"%s\" to %s", cmd, client.ID)
		err = MyCommandStore.AddCommand(id, cmd)
		if err != nil {
			LogInfo.Printf("[!] Failed to store message to %s: %v", id, err)
			return err
		} else {
			LogInfo.Printf("[+] Stored message \"%s\" to %s", cmd, id)
		}
	} else {
		// If not connected, only store the command in the CommandStore
		err := MyCommandStore.AddCommand(id, cmd)
		if err != nil {
			LogInfo.Printf("[!] Failed to store message to %s: %v", id, err)
			return err
		} else {
			LogInfo.Printf("[+] Stored message \"%s\" to %s", cmd, id)
		}
	}
	// Update front-end
	PingWebSocket("command_update", "")
	return nil
}

// Shut down an agent connection
func ShutDownConnection(id string) error {
	ClientsMu.Lock()
	client, exists := Clients[id]
	ClientsMu.Unlock()

	if !client.Reachable {
		return fmt.Errorf("Client %s connection already closed", id)
	}

	// First, check if the client really exists
	if !exists {
		return fmt.Errorf("client with ID '%s' not found", id)
	}

	// If the client is connected, close the running connection
	if client.Conn != nil {
		if err := client.Conn.Close(); err != nil {
			LogInfo.Printf("[!] Failed to close connection for client %s: %v", id, err)
			return fmt.Errorf("failed to close connection for client with ID '%s'", id)
		}
	}

	LogInfo.Printf("[*] Shut down connection for client %s", id)

	markClientUnreachable(id)

	return nil
}

// Delete an agent connection
func DelConnection(id string) error {
	ClientsMu.Lock()
	client, exists := Clients[id]
	ClientsMu.Unlock()

	// First, check if the client really exists
	if !exists {
		return fmt.Errorf("client with ID '%s' not found", id)
	}

	// If the client is connected, close the running connection
	if client.Conn != nil {
		if err := client.Conn.Close(); err != nil {
			LogInfo.Printf("[!] Failed to close connection for client %s: %v", id, err)
		}
	}

	LogInfo.Printf("[-] Client %s deleted", id)

	// Delete client from Clients tab
	ClientsMu.Lock()
	delete(Clients, id)
	ClientsMu.Unlock()

	// Delete commands related to this client in the CommandStore
	MyCommandStore.Lock()
	delete(MyCommandStore.Commands, id)
	if err := MyCommandStore.Save(); err != nil {
		MyCommandStore.Unlock()
		return fmt.Errorf("failed to save MyCommandStore: %v", err)
	}
	MyCommandStore.Unlock()

	// Delete client in the ClientStore
	MyClientStore.Lock()
	storeClient, exists := MyClientStore.Connections[id]
	if exists && storeClient.Conn != nil {
		if err := storeClient.Conn.Close(); err != nil {
			LogInfo.Printf("[!] Failed to close stored connection for client %s: %v", id, err)
		}
	}
	delete(MyClientStore.Connections, id)
	if err := MyClientStore.Save(); err != nil {
		MyClientStore.Unlock()
		return fmt.Errorf("failed to save MyClientStore: %v", err)
	}
	MyClientStore.Unlock()

	/// Update front-end
	PingWebSocket("connection_update", "")

	return nil

}

// Ping front-end when data from back-end changed
func PingWebSocket(msg string, id string) {
	payload := fmt.Sprintf(`{"type":"%s", "id":"%s"}`, msg, id)
	wsmanager.Broadcast(payload)
}

// Retrieve local IP address from the running server machine
func GetLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	for _, iface := range interfaces {
		// Ignore interfaces that are down or loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			// Skip APIPA (169.254.x.x)
			if ip[0] == 169 && ip[1] == 254 {
				continue
			}

			// Return first valid private IP
			if isPrivateIP(ip) {
				return ip.String()
			}
		}
	}

	return "127.0.0.1"
}

// Check if ip is private
func isPrivateIP(ip net.IP) bool {
	return ip.IsPrivate()
}

// Retrieve list from a string with \n character as separator
func SplitLines(s string) []string {
	return strings.Split(s, "\n")
}

// Join strings and separated them with \n character
func JoinLines(lines []string) string {
	return strings.Join(lines, "\n")
}
