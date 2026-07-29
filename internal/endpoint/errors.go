package endpoint

import "errors"

// ErrNilEndpointModel is triggered when an expected model is missing.
var ErrNilEndpointModel = errors.New("endpoint model cannot be nil")

// ErrDanglingReference occurs when a mapping points to an Identity that no longer exists in the OpenAPI spec.
var ErrDanglingReference = errors.New("mapping points to a non-existent identity")

// ErrInvalidResolutionState is an internal guard error.
var ErrInvalidResolutionState = errors.New("relative is in an invalid resolution state")
