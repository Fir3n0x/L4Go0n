package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Handle report execution and save reports in reports directory

// CommandReport structure
type CommandReport struct {
	Timestamp string `json:"timestamp"`
	Command   string `json:"command"`
	Output    string `json:"output"`
}

// Save report for a given client id, command and output result
func SaveReport(clientID string, cmd string, output string) error {

	client, exists := MyClientStore.Connections[clientID]
	if !exists {
		return fmt.Errorf("client %s not found in ClientStore", clientID)
	}
	filename := fmt.Sprintf("reports/%s-%s.json", client.Name, clientID)

	var reports []CommandReport

	// Loading if already exists
	data, err := os.ReadFile(filename)
	if err == nil {
		json.Unmarshal(data, &reports)
	}

	// Add up the new report execution
	reports = append(reports, CommandReport{
		Timestamp: time.Now().Format("2006-01-02 15:04:05.000"),
		Command:   cmd,
		Output:    output,
	})

	// Save in the correct .json file
	updated, _ := json.MarshalIndent(reports, "", "  ")
	return os.WriteFile(filename, updated, 0644)
}
