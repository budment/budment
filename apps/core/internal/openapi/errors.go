package openapi

import "fmt"

// ErrEmptyInput indicates that Parse received an empty OpenAPI document.
var ErrEmptyInput = fmt.Errorf("openapi input is empty")

// UnsupportedVersionError indicates the document is not OpenAPI 3.x.
type UnsupportedVersionError struct {
	Version string
}

// Error implements error.
func (e *UnsupportedVersionError) Error() string {
	return fmt.Sprintf("unsupported OpenAPI version %q: only 3.x is supported", e.Version)
}

// ParseDocumentError indicates low-level document parsing failed.
type ParseDocumentError struct {
	Err error
}

// Error implements error.
func (e *ParseDocumentError) Error() string {
	return fmt.Sprintf("parse OpenAPI document: %v", e.Err)
}

// Unwrap returns the underlying error for errors.Is and errors.As compatibility.
func (e *ParseDocumentError) Unwrap() error {
	return e.Err
}

// ValidateDocumentError indicates OpenAPI schema validation failed.
type ValidateDocumentError struct {
	Err error
}

// Error implements error.
func (e *ValidateDocumentError) Error() string {
	return fmt.Sprintf("validate OpenAPI document: %v", e.Err)
}

// Unwrap returns the underlying error for errors.Is and errors.As compatibility.
func (e *ValidateDocumentError) Unwrap() error {
	return e.Err
}
