package endpoint

import "fmt"

type ResolutionStatus string

const (
	StatusResolved ResolutionStatus = "resolved"
	StatusPending  ResolutionStatus = "pending"
	StatusIgnored  ResolutionStatus = "ignored"
)

type Endpoint struct {
	Protocol   string // e.g., "rest", "grpc", "graphql", "kafka"
	Method     string
	Path       string
	Identities []Identity
	Relatives  []Relative
}

type Identity struct {
	Name     string
	NodePath []string
	Types    []string
	Status   ResolutionStatus
	TargetID string // Valid only for Branch Identities
}

type Relative struct {
	Name     string
	NodePath []string
	Types    []string
	Status   ResolutionStatus
	TargetID string // Points to a RootNode's GlobalID
}

type CandidateCategory string

const (
	CategoryRoot   CandidateCategory = "root"
	CategoryBranch CandidateCategory = "branch"
)

// SemanticContext identifies the closest owning resource (ignoring wrappers like "body" or "data").
type SemanticContext struct {
	ResourceName string // e.g., "users", "categories"
	IsCurrent    bool   // True if ResourceName matches the current API endpoint
}

type IdentityCandidate struct {
	Name           string
	NodePath       []string
	Types          []string
	Context        SemanticContext
	Category       CandidateCategory
	TargetResource string // Valid only if Category is CategoryBranch

	// Origin metadata for resolution
	OriginProtocol string
	OriginMethod   string
	OriginPath     string
	TargetID       string
}

type RelativeCandidate struct {
	Name           string
	NodePath       []string
	Types          []string
	OriginProtocol string
	OriginMethod   string
	OriginPath     string
	TargetID       string
}

type DiscoveryResult struct {
	RootCandidates     []IdentityCandidate
	BranchCandidates   []IdentityCandidate
	RelativeCandidates []RelativeCandidate
}

type IdentityGraph struct {
	Roots    map[string]*RootNode
	Branches map[string]*BranchNode
}

func NewIdentityGraph() *IdentityGraph {
	return &IdentityGraph{
		Roots:    make(map[string]*RootNode),
		Branches: make(map[string]*BranchNode),
	}
}

type RootNode struct {
	Protocol      string
	Method        string
	Resource      string
	SemanticOwner string // Used for smart similarity scoring
	Name          string
	Types         []string
	Children      []*BranchNode
}

// GlobalID: protocol:method:resource:name
func (r *RootNode) GlobalID() string {
	return fmt.Sprintf("%s:%s:%s:%s", r.Protocol, r.Method, r.Resource, r.Name)
}

type BranchNode struct {
	Protocol string
	Method   string
	Resource string
	Name     string
	Parent   *RootNode // Direct reference to the RootNode (prevents circular nesting)
}
