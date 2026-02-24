package cmd

import (
	"encoding/json"
	"os"
	"sync"
)

// Handle preset command store

// Global variables
var MyPresetCommandStore = &PresetCommandStore{
	File: "preset_commands.json",
}

// Preset Command Store
type PresetCommandStore struct {
	sync.Mutex
	File     string
	Commands map[string][]string
}

// Load preset commands from presetCommandStore.json file
func (ps *PresetCommandStore) Load() error {
	ps.Lock()
	defer ps.Unlock()

	if ps.Commands == nil {
		ps.Commands = make(map[string][]string)
	}

	data, err := os.ReadFile(ps.File)
	if err != nil {
		if os.IsNotExist(err) {
			ps.Commands = make(map[string][]string)
			return nil
		}
		return err
	}

	// if file is empty, initialize an empty map
	if len(data) == 0 {
		ps.Commands = make(map[string][]string)
		return nil
	}

	var parsed map[string][]string
    if err := json.Unmarshal(data, &parsed); err != nil {
        return err
    }

    ps.Commands = parsed
    return nil
}

// Save new preset command store into presetCommandStore.json file
func (ps *PresetCommandStore) Save() error {
	data, err := json.MarshalIndent(ps.Commands, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(ps.File, data, 0644)
}

// Add a new preset into the presetCommandStore file
func (ps *PresetCommandStore) AddPreset(presetName string, commands []string) error {
	if err := ps.Load(); err != nil {
        return err
    }
	
	ps.Lock()
	defer ps.Unlock()

	if ps.Commands == nil {
		ps.Commands = make(map[string][]string)
	}

	ps.Commands[presetName] = commands
	err := ps.Save()
	if err != nil {
		return err
	}

	return nil
}

// Delete a preset command
func (ps *PresetCommandStore) DeletePresetCommand(presetName string) error {
	if err := ps.Load(); err != nil {
        return err
    }

	ps.Lock()
	defer ps.Unlock()

	delete(ps.Commands, presetName)

	return ps.Save()
}

// Delete all preset command
func (ps *PresetCommandStore) DeleteAllPresetCommand() {
	ps.Lock()
	defer ps.Unlock()

	for name := range ps.Commands {
		delete(ps.Commands, name)
	}

	_ = ps.Save()
}

// Get commands from a preset
func (ps *PresetCommandStore) getCommandsFromPreset(presetName string) []string {
	if err := ps.Load(); err != nil {
		return nil
	}

	ps.Lock()
	defer ps.Unlock()

	return ps.Commands[presetName]
}
