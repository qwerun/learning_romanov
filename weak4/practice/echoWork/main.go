package main

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"log/slog"
	"net/http"
	"regexp"
)

func main() {
	e := echo.New()

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}

		slog.Error("HTTP error", "status", code, "error", err)

		if code == http.StatusNotFound {
			c.JSON(http.StatusNotFound, map[string]string{
				"error":   "Custom 404: Route Not Found",
				"message": "The requested endpoint does not exist",
			})
			return
		}
		e.DefaultHTTPErrorHandler(err, c)
	}

	e.GET("/", slash)
	e.GET("/hello/:name", hello, ValidateName)

	err := e.Start(":8080")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}

}

func hello(c echo.Context) error {
	n := c.Param("name")
	return c.String(http.StatusOK, fmt.Sprintf("Hello %s!", n))
}

func slash(c echo.Context) error {
	return c.String(http.StatusOK, "Home")
}

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

func ValidateName(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("name")
		if !nameRegex.MatchString(name) {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid name format: only letters and numbers are allowed")
		}
		return next(c)
	}
}
