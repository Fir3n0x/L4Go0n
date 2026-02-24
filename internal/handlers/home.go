package handlers

import (
	"net/http"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// Handle specific endpoints

// Handle dashboard endpoint
func Dashboard(c echo.Context) error {
	sess, _ := session.Get("session", c)
	username := sess.Values["username"]
	return c.Render(http.StatusOK, "dashboard.html", map[string]interface{}{
		"Title":    "L4Go0n",
		"Username": username,
	})
}

// Handle Login endpoint
func Login(c echo.Context) error {
	return c.Render(http.StatusUnauthorized, "login.html", map[string]interface{}{
		"Error": "Wrong credentials",
	})
}
