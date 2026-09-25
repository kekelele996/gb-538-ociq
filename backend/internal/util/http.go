package util

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return e.Message + ": " + e.Cause.Error()
}

func (e *AppError) Unwrap() error { return e.Cause }

func NewError(status int, code, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Cause: cause}
}

func Validation(message string, cause error) *AppError {
	return NewError(http.StatusUnprocessableEntity, "validation_failed", message, cause)
}

func Conflict(message string, cause error) *AppError {
	return NewError(http.StatusConflict, "conflict", message, cause)
}

func NotFound(message string) *AppError {
	return NewError(http.StatusNotFound, "not_found", message, nil)
}

func Forbidden(message string) *AppError {
	return NewError(http.StatusForbidden, "forbidden", message, nil)
}

func Respond(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{
		"code":       "ok",
		"message":    "success",
		"data":       data,
		"request_id": c.GetString("request_id"),
	})
}

func RespondError(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.Status, gin.H{
			"code": appErr.Code, "message": appErr.Message,
			"request_id": c.GetString("request_id"),
		})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"code": "internal_error", "message": "服务暂时无法完成请求",
		"request_id": c.GetString("request_id"),
	})
}
