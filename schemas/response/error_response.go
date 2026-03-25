package response

import "github.com/gofiber/fiber/v2"

// ErrorResponse represents the standard JSON error structure for the entire project.
// Example: {"error": "Invalid token"}
type ErrorResponse struct {
	Error string `json:"error"`
}

// SendError is a helper dedicated to returning standardized error responses.
// Use this throughout the project to maintain architectural consistency.
func SendError(c *fiber.Ctx, statusCode int, message string) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Error: message,
	})
}

// SendMessage is a helper for returning simple success messages in JSON.
func SendMessage(c *fiber.Ctx, statusCode int, message string) error {
	return c.Status(statusCode).JSON(fiber.Map{
		"message": message,
	})
}
