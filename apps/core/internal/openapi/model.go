package openapi

// Model represents a strictly deterministic, flattened API structure.
// This is the Single Source of Truth for the Parse phase.
type Model struct {
	OpenAPIVersion string
	Info           Info
	Servers        []Server
	Operations     []Operation
}

type Info struct {
	Title       string
	Description string
	Version     string
}

type Server struct {
	URL         string
	Description string
}

// Operation represents an API endpoint, strictly ordered by path depth and method priority.
type Operation struct {
	Path        string
	Method      string
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool

	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   []Response
	Security    []SecurityRequirement
}

type Parameter struct {
	Name        string
	In          string
	Required    bool
	Description string
	Deprecated  bool
	Schema      *Schema
}

type RequestBody struct {
	Description string
	Required    bool
	Contents    []Content
}

type Response struct {
	StatusCode  string
	Description string
	Headers     []Header
	Contents    []Content
}

type Header struct {
	Name        string
	Description string
	Required    bool
	Deprecated  bool
	Schema      *Schema
}

type Content struct {
	MediaType string
	Schema    *Schema
}

type SecurityRequirement struct {
	Schemes []SecuritySchemeRequirement
}

type SecuritySchemeRequirement struct {
	Name   string
	Scopes []string
}

type Schema struct {
	Ref                  string
	Types                []string
	Format               string
	Description          string
	Nullable             bool
	Required             []string
	Enum                 []string
	Properties           []SchemaProperty
	Items                *Schema
	OneOf                []*Schema
	AnyOf                []*Schema
	AllOf                []*Schema
	AdditionalProperties *Schema
}

type SchemaProperty struct {
	Name   string
	Schema *Schema
}
