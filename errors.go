package bearing

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"modernc.org/sqlite"
)

type Error struct {
	Code           string `json:"code"`
	Message        string `json:"message"`
	Retryable      bool   `json:"retryable"`
	RequestID      string `json:"request_id,omitempty"`
	CurrentVersion int    `json:"current_version,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func fail(code, message string) *Error { return &Error{Code: code, Message: message} }

func ErrorDetails(err error) *Error {
	var domain *Error
	if errors.As(err, &domain) {
		return domain
	}
	var se *sqlite.Error
	if errors.As(err, &se) && (se.Code()&255 == 5 || se.Code()&255 == 6) {
		return &Error{Code: "busy", Message: "Another write is running. Retry with the same request ID.", Retryable: true}
	}
	return fail("io_error", err.Error())
}

func NewID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}

// DecodeInput rejects ambiguous or misspelled input for CLI and MCP alike.
func DecodeInput(data []byte, target any) error {
	if trimmed := strings.TrimSpace(string(data)); !strings.HasPrefix(trimmed, "{") {
		return fail("invalid_input", "Input must be a JSON object.")
	}
	if len(data) > 65536 {
		return fail("invalid_input", "Input exceeds 64 KiB. Keep conclusions and reference the evidence.")
	}
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return fail("invalid_input", fmt.Sprintf("Invalid input: %v", err))
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fail("invalid_input", "Provide one JSON object.")
	}
	return nil
}

func inputSize(input any, limit int) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	if len(data) > limit {
		return fail("invalid_input", fmt.Sprintf("Input exceeds %d bytes. Keep conclusions and reference the evidence.", limit))
	}
	return nil
}
