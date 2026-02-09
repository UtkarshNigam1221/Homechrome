// Package response provides HTTP response utilities
package response

import (
	"encoding/json"
	"net/http"

	"github.com/handloom/admin/pkg/errors"
)

// Response is a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorBody represents an error in the response
type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Meta contains pagination and other metadata
type Meta struct {
	Limit      int    `json:"limit,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// JSON sends a JSON response
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Success sends a successful response
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMeta sends a successful response with metadata
func SuccessWithMeta(w http.ResponseWriter, data interface{}, meta *Meta) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Created sends a 201 Created response
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// NoContent sends a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error sends an error response
func Error(w http.ResponseWriter, err error) {
	var status int
	var body ErrorBody

	if appErr, ok := errors.AsAppError(err); ok {
		status = appErr.HTTPStatus
		body = ErrorBody{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Details: appErr.Details,
		}
	} else {
		status = http.StatusInternalServerError
		body = ErrorBody{
			Code:    string(errors.ErrCodeInternal),
			Message: "An internal error occurred",
		}
	}

	JSON(w, status, Response{
		Success: false,
		Error:   &body,
	})
}

// BadRequest sends a 400 Bad Request response
func BadRequest(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, Response{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrCodeBadRequest),
			Message: message,
		},
	})
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(w http.ResponseWriter, message string) {
	JSON(w, http.StatusUnauthorized, Response{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrCodeUnauthorized),
			Message: message,
		},
	})
}

// Forbidden sends a 403 Forbidden response
func Forbidden(w http.ResponseWriter, message string) {
	JSON(w, http.StatusForbidden, Response{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrCodeForbidden),
			Message: message,
		},
	})
}

// NotFound sends a 404 Not Found response
func NotFound(w http.ResponseWriter, resource string) {
	JSON(w, http.StatusNotFound, Response{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrCodeNotFound),
			Message: resource + " not found",
		},
	})
}

// ValidationError sends a validation error response
func ValidationError(w http.ResponseWriter, details interface{}) {
	JSON(w, http.StatusBadRequest, Response{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrCodeValidation),
			Message: "Validation failed",
			Details: details,
		},
	})
}

// InternalError sends a 500 Internal Server Error response
func InternalError(w http.ResponseWriter) {
	JSON(w, http.StatusInternalServerError, Response{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrCodeInternal),
			Message: "An internal error occurred",
		},
	})
}
