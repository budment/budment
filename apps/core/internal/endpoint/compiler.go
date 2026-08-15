package endpoint

import (
	"log/slog"
	"sort"

	"github.com/vunas/blaster/internal/schema"
)

type Compiler struct {
	sync       *Synchronizer
	discoverer *Discoverer
	resolver   *Resolver
	log        *slog.Logger
}

func NewCompiler(s *Synchronizer, d *Discoverer, r *Resolver, log *slog.Logger) *Compiler {
	return &Compiler{sync: s, discoverer: d, resolver: r, log: log}
}

func (c *Compiler) Compile(actions []schema.Action, oldEndpoints []*Endpoint) []*Endpoint {
	c.log.Info("Ensuring deterministic state (Phase 0: Protocol-Agnostic Sort)")

	// Sort operations to ensure deterministic discovery order
	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].Depth != actions[j].Depth {
			return actions[i].Depth < actions[j].Depth
		}
		if actions[i].Resource != actions[j].Resource {
			return actions[i].Resource < actions[j].Resource
		}
		if actions[i].Priority != actions[j].Priority {
			return actions[i].Priority < actions[j].Priority
		}
		return actions[i].Identifier < actions[j].Identifier
	})

	c.log.Info("Starting Phase 1: Endpoint Reconciliation")
	workingState := c.sync.Reconcile(oldEndpoints, actions)

	c.log.Info("Starting Phase 2: Candidate Discovery")
	discoveryResult := c.discoverer.Discover(actions, workingState)

	c.log.Info("Starting Phase 3: Build Graph and Resolve Branches")
	graph := c.resolver.BuildGraphAndResolveBranches(&discoveryResult, workingState)

	c.log.Info("Starting Phase 4: Resolve Relatives")
	c.resolver.ResolveRelatives(&discoveryResult, graph)

	c.log.Info("Phase 5: Building Final Endpoints for Output")
	return c.buildEndpoints(actions, discoveryResult, workingState, graph)
}

func (c *Compiler) buildEndpoints(actions []schema.Action, result DiscoveryResult, state *WorkingState, graph *IdentityGraph) []*Endpoint {
	var endpoints []*Endpoint
	epMap := make(map[string]*Endpoint)

	for _, action := range actions {
		epMap[action.Identifier] = &Endpoint{
			Protocol: action.Protocol,
			Method:   action.Method,
			Path:     action.Resource,
		}
		endpoints = append(endpoints, epMap[action.Identifier])
	}

	slotIdentity := func(cand IdentityCandidate) {
		epKey := cand.OriginProtocol + ":" + cand.OriginMethod + ":" + cand.OriginPath
		if ep, exists := epMap[epKey]; exists {
			ident := Identity{
				Name:     cand.Name,
				NodePath: cand.NodePath,
				Types:    cand.Types,
				Status:   StatusResolved,
				TargetID: cand.TargetID,
			}

			if cand.TargetID == "" && cand.Category != CategoryRoot {
				ident.Status = StatusPending
			}

			if state.IsIgnored(epKey, cand.NodePath) {
				ident.Status = StatusIgnored
			} else if lockedID, ok := state.GetLockedIdentity(epKey, cand.NodePath); ok {
				if lockedID.TargetID != "" {
					if _, isValid := graph.Roots[lockedID.TargetID]; isValid {
						ident.Status = lockedID.Status
						ident.TargetID = lockedID.TargetID
					} else {
						ident.Status = StatusPending
						ident.TargetID = ""
					}
				} else {
					ident.Status = lockedID.Status
					ident.TargetID = ""
				}
			}

			ep.Identities = append(ep.Identities, ident)
		}
	}

	for _, root := range result.RootCandidates {
		slotIdentity(root)
	}
	for _, branch := range result.BranchCandidates {
		slotIdentity(branch)
	}

	for _, cand := range result.RelativeCandidates {
		epKey := cand.OriginProtocol + ":" + cand.OriginMethod + ":" + cand.OriginPath
		if ep, exists := epMap[epKey]; exists {
			rel := Relative{
				Name:     cand.Name,
				NodePath: cand.NodePath,
				Types:    cand.Types,
				Status:   StatusResolved,
				TargetID: cand.TargetID,
			}

			if cand.TargetID == "" {
				rel.Status = StatusPending
			}

			if state.IsIgnored(epKey, cand.NodePath) {
				rel.Status = StatusIgnored
			} else if lockedRel, ok := state.GetLockedRelative(epKey, cand.NodePath); ok {
				if lockedRel.TargetID != "" {
					if _, isValid := graph.Roots[lockedRel.TargetID]; isValid {
						rel.Status = lockedRel.Status
						rel.TargetID = lockedRel.TargetID
					} else {
						rel.Status = StatusPending
						rel.TargetID = ""
					}
				}
			}

			ep.Relatives = append(ep.Relatives, rel)
		}
	}

	return endpoints
}
