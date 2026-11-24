package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxJSONBodySize = 1 << 20

var (
	errInvalidJSON     = errors.New("invalid JSON request")
	errRequestTooLarge = errors.New("request body is too large")
)

func decodeJSON(writer http.ResponseWriter, request *http.Request, destination any) error {
	if request.Body == nil {
		return errInvalidJSON
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, maxJSONBodySize))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return classifyJSONError(err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errInvalidJSON
		}
		return classifyJSONError(err)
	}
	return nil
}

func classifyJSONError(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return errRequestTooLarge
	}
	return fmt.Errorf("%w: %v", errInvalidJSON, err)
}

func writeJSON(writer http.ResponseWriter, status int, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_, err = writer.Write(append(encoded, '\n'))
	return err
}
