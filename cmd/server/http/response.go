package http

import (
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

// Error sends an error response with the given error
func Error(c *gin.Context, status int, err error) {
	var msg string
	if err != nil {
		msg = err.Error()
	} else {
		msg = "Unknown error"
	}

	c.JSON(status, ErrorResponse{
		Error: msg,
	})

	c.Abort()
}
