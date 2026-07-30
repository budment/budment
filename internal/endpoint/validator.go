package endpoint

import (
	"log/slog"
)

type Validator struct {
	log *slog.Logger
}

func NewValidator(log *slog.Logger) *Validator {
	return &Validator{log: log}
}

func (v *Validator) Validate(endpoints []*Endpoint) int {
	validRoots := make(map[string]bool)
	identitiesPerEndpoint := make(map[string]bool)
	issuesCount := 0

	for _, ep := range endpoints {
		epKey := ep.Protocol + ":" + ep.Method + ":" + ep.Path

		for i := range ep.Identities {
			id := &ep.Identities[i]
			idGlobalKey := epKey + ":" + id.Name

			if identitiesPerEndpoint[idGlobalKey] {
				v.log.Warn("Duplicate identity detected in same endpoint", "endpoint", epKey, "identity", id.Name)
				issuesCount++
			}
			identitiesPerEndpoint[idGlobalKey] = true

			if id.Status == StatusResolved && id.TargetID == "" {
				validRoots[idGlobalKey] = true
			}
		}
	}

	for _, ep := range endpoints {
		epKey := ep.Protocol + ":" + ep.Method + ":" + ep.Path

		for i := range ep.Identities {
			id := &ep.Identities[i]
			if id.Status == StatusResolved && id.TargetID != "" {
				if !validRoots[id.TargetID] {
					v.log.Warn("Orphan Branch detected! Downgrading to Pending.",
						"endpoint", epKey, "branch", id.Name, "invalid_target", id.TargetID)
					id.Status = StatusPending
					id.TargetID = ""
					issuesCount++
				}
			}
		}

		for i := range ep.Relatives {
			rel := &ep.Relatives[i]
			if rel.Status == StatusResolved && rel.TargetID != "" {
				if !validRoots[rel.TargetID] {
					v.log.Warn("Invalid Relative Reference detected! Downgrading to Pending.",
						"endpoint", epKey, "relative", rel.Name, "invalid_target", rel.TargetID)
					rel.Status = StatusPending
					rel.TargetID = ""
					issuesCount++
				}
			}
		}
	}

	if issuesCount > 0 {
		v.log.Warn("Workspace validation completed with issues intercepted.", "issues_fixed", issuesCount)
	} else {
		v.log.Debug("Workspace validation passed perfectly. Semantic graph is consistent.")
	}

	return issuesCount
}
