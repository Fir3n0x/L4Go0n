package cmd

import (
	"encoding/json"
	"os"
	"sync"
)

// Handle CommandStore: Link client Id with its command list

// Global variable
var MyCommandStore = &CommandStore{
	File: "commands.json",
}

// CommandStore structure
type CommandStore struct {
	sync.Mutex
	File      string
	Commands  map[string][]string
	listeners map[string][]chan string
}

// Load commands from commands.json file
func (cs *CommandStore) Load() error {
	cs.Lock()
	defer cs.Unlock()

	cs.Commands = make(map[string][]string)

	data, err := os.ReadFile(cs.File)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &cs.Commands)
}

// Save new CommandStore into commands.json file
func (cs *CommandStore) Save() error {
	data, err := json.MarshalIndent(cs.Commands, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cs.File, data, 0644)
}

// Add command into an Id's list
func (cs *CommandStore) AddCommand(clientID string, command string) error {
	cs.Lock()
	defer cs.Unlock()

	if cs.Commands == nil {
		cs.Commands = make(map[string][]string)
	}

	cs.Commands[clientID] = append(cs.Commands[clientID], command)
	err := cs.Save()
	if err != nil {
		return err
	}

	// Notify listeners
	for _, ch := range cs.listeners[clientID] {
		select {
		case ch <- command:
		default:
		}
	}
	return nil
}

// Delete ONE command from an Id's list
func (cs *CommandStore) DeleteCommand(clientID, command string) {
	cs.Lock()
	defer cs.Unlock()

	cmds := cs.Commands[clientID]
	for i, c := range cmds {
		if c == command {
			cs.Commands[clientID] = append(cmds[:i], cmds[i+1:]...)
			break
		}
	}
	_ = cs.Save()
}

// Delete ALL commands from an Id's list
func (cs *CommandStore) DeleteAllCommands(clientID string) {
	cs.Lock()
	defer cs.Unlock()

	delete(cs.Commands, clientID)
	_ = cs.Save()
}

// Get all commands from an Id's list and delete them from it
func (cs *CommandStore) GetCommands(clientID string) []string {
	if err := cs.Load(); err != nil {
		return nil
	}

	cs.Lock()
	defer cs.Unlock()
	cmds := cs.Commands[clientID]
	delete(cs.Commands, clientID)
	_ = cs.Save()
	return cmds
}

// Create another channel linked to a client Id
func (cs *CommandStore) AddListener(clientID string, ch chan string) {
	cs.Lock()
	defer cs.Unlock()
	if cs.listeners == nil {
		cs.listeners = make(map[string][]chan string)
	}
	cs.listeners[clientID] = append(cs.listeners[clientID], ch)
}

// Remove a listener from the listener's list
func (cs *CommandStore) RemoveListener(clientID string, ch chan string) {
	cs.Lock()
	defer cs.Unlock()
	list := cs.listeners[clientID]
	for i, l := range list {
		if l == ch {
			cs.listeners[clientID] = append(list[:i], list[i+1:]...)
			break
		}
	}
}

// Check whether the command list for a given client is empty or not
func (cs *CommandStore) IsCommandEmpty(clientID string) bool {
	return len(cs.Commands) == 0
}
