package httpapp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/VatsalP117/learnloom/internal/store"
)

type problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeProblem(
	response http.ResponseWriter,
	status int,
	code, message string,
) {
	writeJSON(response, status, problem{Code: code, Message: message})
}

func decodeJSON(
	response http.ResponseWriter,
	request *http.Request,
	maximum int64,
	value any,
) bool {
	request.Body = http.MaxBytesReader(response, request.Body, maximum)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		code := "invalid_json"
		message := "The JSON request body is invalid."
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			code = "request_too_large"
			message = "The request body is too large."
		}
		writeProblem(response, http.StatusBadRequest, code, message)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeProblem(response, http.StatusBadRequest, "invalid_json", "The JSON request body must contain one object.")
		return false
	}
	return true
}

func writeStoreError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeProblem(response, http.StatusNotFound, "not_found", "The requested resource was not found.")
	case errors.Is(err, store.ErrConflict):
		writeProblem(response, http.StatusConflict, "conflict", "The request conflicts with current state.")
	case errors.Is(err, store.ErrForbidden):
		writeProblem(response, http.StatusForbidden, "forbidden", "The request is not allowed.")
	case errors.Is(err, store.ErrQuotaExceeded):
		writeProblem(response, http.StatusTooManyRequests, "quota_exceeded", "The Account quota has been reached.")
	default:
		writeProblem(response, http.StatusBadRequest, "invalid_request", err.Error())
	}
}

func methodNotAllowed(response http.ResponseWriter, methods ...string) {
	for _, method := range methods {
		response.Header().Add("Allow", method)
	}
	writeProblem(response, http.StatusMethodNotAllowed, "method_not_allowed", "The request method is not allowed.")
}
