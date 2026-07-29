package endpoint

import (
	"strings"

	"github.com/vunas/blaster/internal/openapi"
)

// WorkingState holds configurations explicitly set by the developer or resolved in the past.
// It acts as the ultimate firewall against automatic overwrites.
type WorkingState struct {
	LockedIdentities map[string]Identity
	LockedRelatives  map[string]Relative
	IgnoredNodes     map[string]bool
}

func NewWorkingState() *WorkingState {
	return &WorkingState{
		LockedIdentities: make(map[string]Identity),
		LockedRelatives:  make(map[string]Relative),
		IgnoredNodes:     make(map[string]bool),
	}
}

func (w *WorkingState) makeKey(epKey string, nodePath []string) string {
	return epKey + "|" + strings.Join(nodePath, ".")
}

func (w *WorkingState) IsIgnored(epKey string, nodePath []string) bool {
	return w.IgnoredNodes[w.makeKey(epKey, nodePath)]
}

func (w *WorkingState) GetLockedIdentity(epKey string, nodePath []string) (Identity, bool) {
	id, ok := w.LockedIdentities[w.makeKey(epKey, nodePath)]
	return id, ok
}

func (w *WorkingState) GetLockedRelative(epKey string, nodePath []string) (Relative, bool) {
	rel, ok := w.LockedRelatives[w.makeKey(epKey, nodePath)]
	return rel, ok
}

// Endpoint Reconciliation.
type Synchronizer struct{}

func NewSynchronizer() *Synchronizer {
	return &Synchronizer{}
}

// Reconcile extracts strictly valid, user-modified states from existing YAMLs.
func (s *Synchronizer) Reconcile(oldEndpoints []*Endpoint, apiModel *openapi.Model) *WorkingState {
	state := NewWorkingState()

	validOps := make(map[string]string, len(apiModel.Operations))
	for _, op := range apiModel.Operations {
		lookupKey := strings.ToUpper(op.Method) + ":" + strings.TrimSuffix(op.Path, "/")
		validOps[lookupKey] = op.Path
	}

	for _, oldEp := range oldEndpoints {
		lookupKey := strings.ToUpper(oldEp.Method) + ":" + strings.TrimSuffix(oldEp.Path, "/")

		canonicalPath, exists := validOps[lookupKey]
		if !exists {
			continue
		}

		epKey := strings.ToUpper(oldEp.Method) + ":" + canonicalPath

		for _, rel := range oldEp.Relatives {
			stateKey := state.makeKey(epKey, rel.NodePath)
			if rel.Status == StatusIgnored {
				state.IgnoredNodes[stateKey] = true
			} else if rel.Status == StatusResolved && rel.TargetID != "" {
				state.LockedRelatives[stateKey] = rel
			}
		}

		for _, id := range oldEp.Identities {
			stateKey := state.makeKey(epKey, id.NodePath)
			switch id.Status {
			case StatusIgnored:
				state.IgnoredNodes[stateKey] = true
			case StatusResolved:
				state.LockedIdentities[stateKey] = id
			}
		}
	}

	return state
}
