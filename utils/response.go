package utils

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type APIError struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func Success(c *gin.Context, statusCode int, data interface{}, message string) {
	c.JSON(statusCode, APIResponse{Success: true, Message: message, Data: data})
}

func ErrorRes(c *gin.Context, statusCode int, errCode, message string) {
	c.JSON(statusCode, APIError{Success: false, Error: errCode, Message: message})
}
