package schema

// Action represents a generic protocol operation (REST Endpoint, gRPC Method, Kafka Topic).
type Action struct {
	Protocol   string // e.g., "rest", "grpc", "kafka"
	Method     string // e.g., "GET", "POST", "EVENT"
	Identifier string // e.g., "rest:POST:/users", "grpc:UserService/Create"
	Resource   string // e.g., "/users", "UserService", "user.events"

	// Delegating sorting responsibilities to the Plugin ensures the Core remains agnostic.
	Depth    int // Determines path depth (shallow vs deep)
	Priority int // Determines execution priority (e.g., POST=1, GET=2 for REST)

	Inputs  []Field // Request Parameters, Body, or Kafka Payload
	Outputs []Field // Response Body, Headers (2xx only for REST)
}

// Field represents a flattened, deterministic data node.
type Field struct {
	Name     string
	Path     []string // e.g., ["body", "user", "id"]
	Types    []string
	Format   string // Important for semantic discovery (e.g., "uuid")
	IsHeader bool
}

// ParserPlugin is the contract that OpenAPI, gRPC, and Kafka parsers must implement.
type ParserPlugin interface {
	Parse(specData []byte) ([]Action, error)
}
