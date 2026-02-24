package handlers

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"
)

// File to handle creation of agent

// Structure used to store data given to create an agent
type AgentConfig struct {
	ID          string
	IP_SERVER   string
	PORT_SERVER string
	TYPE        string
	OS          string
	ServerPublicKey []byte
}

// Generate an agent regarding given AgentConfig structure
func GenerateAgentSource(config AgentConfig) ([]byte, error) {
	path := filepath.Join("internal", "storage", "client", config.TYPE, config.OS, "main.c")
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		fmt.Printf("Problème path %s\n", path)
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

//docker run --rm -v %cd%:/src -w /src gcc:latest gcc source.c -o agent.bin
func GetBuildConfig(os, agentType, icon string, sourcePath, exePath, iconPath string) []string {
	var args []string

	switch os {
	case "windows":
		args = []string{sourcePath, "-o", exePath, "-lws2_32"}
		if agentType != "simple" {
			args = append(args, "-mwindows")
		}
		if icon != "none" {
			args = append([]string{sourcePath, iconPath}, args[1:]...)
		}
	case "linux":
		args = []string{sourcePath, "-o", exePath}
	default:
		args = []string{sourcePath, "-o", exePath}
	}

	return args
}
