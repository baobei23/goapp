package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type BaseResponse struct {
	Data any `json:"data,omitempty"`
	Meta any `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Something went wrong"`
}

// JSON sends a JSON response with the given data and meta
func JSON(c *gin.Context, status int, data any, meta any) {
	c.JSON(status, BaseResponse{
		Data: data,
		Meta: meta,
	})
}

// Error sends a sanitized error response and logs internal server errors
func Error(c *gin.Context, status int, err error) {
	var clientMsg string

	if status >= http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "internal server error",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"error", err,
		)
		clientMsg = "Internal server error. Please try again later."
	} else {
		if err != nil {
			clientMsg = err.Error()
		} else {
			clientMsg = "Bad request"
		}
	}

	c.JSON(status, ErrorResponse{
		Error: clientMsg,
	})

	c.Abort()
}
