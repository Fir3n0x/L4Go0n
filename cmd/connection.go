package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Handle ClientStore: Link client Id with their info

// Global variables
var MyClientStore = &ClientStore{
	File:        "connections.json",
	Connections: make(map[string]Client),
}

// ClientStore structure
type ClientStore struct {
	sync.Mutex
	File        string
	Connections map[string]Client
}

// Get the Client object from the connection list (using the client's id)
func (cl *ClientStore) GetClient(ID string) (Client, bool) {
	cl.Lock()
	defer cl.Unlock()

	client, exists := cl.Connections[ID]
	return client, exists
}

// Set new values for a given Client
func (cl *ClientStore) SetClient(ID string, updated Client) error {
	cl.Lock()
	defer cl.Unlock()

	if _, exists := cl.Connections[ID]; !exists {
		return fmt.Errorf("client with ID %s not found", ID)
	}

	cl.Connections[ID] = updated
	return cl.Save()
}

// Load client Connections from connections.json file
func (cl *ClientStore) Load() error {
	cl.Lock()
	defer cl.Unlock()

	data, err := os.ReadFile(cl.File)
	if err != nil {
		if os.IsNotExist(err) {
			cl.Connections = make(map[string]Client)
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &cl.Connections)
}

// Save new Connections in connections.json file
func (cl *ClientStore) Save() error {
	data, err := json.MarshalIndent(cl.Connections, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(cl.File, data, 0644)
}

// Add a Client in Connections tab and save it in connections.json file
func (cl *ClientStore) AddClient(in_c Client) error {
	cl.Lock()
	defer cl.Unlock()

	clientId := in_c.ID

	if _, exists := cl.Connections[clientId]; !exists {
		cl.Connections[clientId] = in_c
		return cl.Save()
	} else if in_c.Reachable != cl.Connections[clientId].Reachable {
		client := cl.Connections[clientId]
		client.Reachable = in_c.Reachable
		client.ConnServer = in_c.ConnServer
		client.ConnProxy = in_c.ConnProxy
		cl.Connections[clientId] = client

		return cl.Save()
	}

	return nil
}

// Reset default network value for ALL clients
func (cl *ClientStore) ResetConn() error {
	cl.Lock()
	defer cl.Unlock()

	for id, client := range cl.Connections {
		client.Reachable = false
		cl.Connections[id] = client
	}
	return cl.Save()
}

// Check whether a client is in the Connections tab or not
func (cl *ClientStore) IsInClientStore(ID string) bool {
	cl.Lock()
	defer cl.Unlock()

	for id := range cl.Connections {
		if ID == id {
			return true
		}
	}

	return false
}
