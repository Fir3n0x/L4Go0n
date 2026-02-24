package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// Handle report files and data

// Retrieve the content of a report execution file
func GetReportHandler(c echo.Context) error {
	clientID := c.QueryParam("id")
	filename := fmt.Sprintf("reports/%s.json", clientID)

	data, err := os.ReadFile(filename)
	if err != nil {
		return c.String(http.StatusNotFound, "Report not found")
	}

	return c.JSONBlob(http.StatusOK, data)
}

// Retrieve all the report execution files
func GetReportsListHandler(c echo.Context) error {
	dir := "reports"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Impossible to read report folder")
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}

	return c.JSON(http.StatusOK, files)
}
