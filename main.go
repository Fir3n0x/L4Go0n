package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/Fir3n0x/my-c2-dashboard/cmd"
	"github.com/Fir3n0x/my-c2-dashboard/internal/routes"
	wsmanager "github.com/Fir3n0x/my-c2-dashboard/ws"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var upgrader = websocket.Upgrader{}

func main() {
	// Init data (Log, Command, Client)
	initLogging()
	initCommandStore()
	initClientStore()

	// Start web server
	e := setupWebServer()
	go func() {
		e.Logger.Fatal(e.Start("0.0.0.0:8080"))
	}()

	// Start C2 server
	startC2Server()
}

func setupWebServer() *echo.Echo {
	e := echo.New()

	// WebSocket Back <--> Front
	e.GET("/ws", func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		wsmanager.ClientsMu.Lock()
		wsmanager.Clients[conn] = true
		wsmanager.ClientsMu.Unlock()

		// Loop to maintain an open connection
		go func() {
			defer func() {
				wsmanager.ClientsMu.Lock()
				delete(wsmanager.Clients, conn)
				wsmanager.ClientsMu.Unlock()
				conn.Close()
			}()

			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()

		return nil
	})

	// WebServer
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret-key"))))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, _ := session.Get("session", c)
			sess.Options = &sessions.Options{
				Path:     "/",
				MaxAge:   86400 * 1,
				HttpOnly: true,
				Secure:   false,
			}
			return next(c)
		}
	})

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Renderer = cmd.TempRenderer

	routes.Register(e)

	e.Static("/static", "static")

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if he, ok := err.(*echo.HTTPError); ok && he.Code == http.StatusNotFound {
			_ = c.Redirect(http.StatusSeeOther, "/")
			return
		}
		c.String(http.StatusInternalServerError, "Something went wrong")
	}

	return e
}

func startC2Server() {
	// Start server
	address := net.JoinHostPort(cmd.IPServer, cmd.PortServer)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Server started on %s\n", address)

	// Launch proxies
	for _, port := range cmd.ProxyPorts {
		go cmd.StartProxy(port)
	}

	// Accept connection on main port (5437 localhost)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection on C2 port: %v", err)
			continue
		}
		go cmd.HandleConnection(conn)
	}
}

func initLogging() {
	// Handle log file
	logFile, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	cmd.LogInfo = log.New(logFile, "INFO : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	log.SetOutput(logFile)
}

func initCommandStore() {
	// Initialize command store
	err := cmd.MyCommandStore.Load()
	if err != nil {
		log.Fatalf("Failed to load command store: %v", err)
	}
}

func initClientStore() {
	// Initialize client store
	err := cmd.MyClientStore.Load()
	if err != nil {
		log.Fatalf("Failed to load client store: %v", err)
	}

	err = cmd.MyClientStore.ResetConn()
	if err != nil {
		log.Fatalf("Failed to reset connection in client store: %v", err)
	}

	// Fill up ClientStore with loaded data from connections.json
	// Initialize structure
	for id, c := range cmd.MyClientStore.Connections {
		cmd.Clients[id] = &cmd.Client{
			Name:          c.Name,
			ID:            c.ID,
			IP:            c.IP,
			OS:            c.OS,
			SrcPort:       c.SrcPort,
			DstPort:       c.DstPort,
			LastConnexion: c.LastConnexion,
			TYPE:          c.TYPE,
			ConnServer:    nil,
			ConnProxy:     nil,
			Icon:          c.Icon,
			Reachable:     false,
		}
	}
}
