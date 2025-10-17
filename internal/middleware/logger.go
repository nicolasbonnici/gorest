package middleware

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

func HTTPLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		logRequest(c, start)

		err := c.Next()

		logResponse(c, start)

		return err
	}
}


func logRequest(c *fiber.Ctx, start time.Time) {
	fmt.Fprintf(os.Stdout, "[%s] --> %s %s %s\n",
		start.Format("2006-01-02 15:04:05"),
		c.Method(),
		c.Path(),
		c.Protocol(),
	)

	if len(c.Queries()) > 0 {
		fmt.Fprintf(os.Stdout, "    Query: %v\n", c.Queries())
	}

	fmt.Fprintf(os.Stdout, "    From: %s\n", c.IP())
	os.Stdout.Sync()
}

func logResponse(c *fiber.Ctx, start time.Time) {
	duration := time.Since(start)
	status := c.Response().StatusCode()

	symbol := getStatusSymbol(status)

	fmt.Fprintf(os.Stdout, "[%s] <-- %s %s %s [%d] %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		symbol,
		c.Method(),
		c.Path(),
		status,
		duration,
	)

	bodySize := len(c.Response().Body())
	if bodySize > 0 {
		fmt.Fprintf(os.Stdout, "    Size: %s\n", formatBytes(bodySize))
	}
	os.Stdout.Sync()
}

func getStatusSymbol(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "✓" // Success
	case status >= 300 && status < 400:
		return "→" // Redirect
	case status >= 400 && status < 500:
		return "✗" // Client error
	case status >= 500:
		return "!" // Server error
	default:
		return "?"
	}
}

func formatBytes(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
	)

	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
