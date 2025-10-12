package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success returns a success response
func Success(data interface{}) Response {
	return Response{
		Success: true,
		Message: "success",
		Data:    data,
	}
}

// Error returns an error response
func Error(statusCode int, message string) Response {
	return Response{
		Success: false,
		Message: message,
	}
}

// JSON sends a JSON response
func JSON(c *gin.Context, statusCode int, response Response) {
	c.JSON(statusCode, response)
}

// SuccessJSON sends a success JSON response
func SuccessJSON(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, Success(data))
}

// ErrorJSON sends an error JSON response
func ErrorJSON(c *gin.Context, statusCode int, message string) {
	JSON(c, statusCode, Error(statusCode, message))
}
