package endpoint

import (
	"log/slog"
	"sort"
	"strings"

	"github.com/vunas/blaster/internal/openapi"
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

func (c *Compiler) Compile(apiModel *openapi.Model, oldEndpoints []*Endpoint) []*Endpoint {
	c.log.Info("Ensuring deterministic state (Phase 0: Sort & Flatten)")

	// Sort operations to ensure deterministic discovery order
	sort.SliceStable(apiModel.Operations, func(i, j int) bool {
		depthI := strings.Count(apiModel.Operations[i].Path, "/")
		depthJ := strings.Count(apiModel.Operations[j].Path, "/")

		// Sort by depth (shallow paths first)
		if depthI != depthJ {
			return depthI < depthJ
		}

		// Sort alphabetically if depths are equal
		if apiModel.Operations[i].Path != apiModel.Operations[j].Path {
			return apiModel.Operations[i].Path < apiModel.Operations[j].Path
		}

		// Sort by HTTP method as a final tie-breaker (e.g., GET before POST)
		return apiModel.Operations[i].Method < apiModel.Operations[j].Method
	})

	c.log.Info("Starting Phase 1: Endpoint Reconciliation")
	workingState := c.sync.Reconcile(oldEndpoints, apiModel)

	c.log.Info("Starting Phase 2: Candidate Discovery (Root, Branch, Relative)")
	discoveryResult := c.discoverer.Discover(apiModel, workingState)

	c.log.Info("Starting Phase 3: Build Graph and Resolve Branches")
	graph := c.resolver.BuildGraphAndResolveBranches(&discoveryResult, workingState)

	c.log.Info("Starting Phase 4: Resolve Relatives")
	c.resolver.ResolveRelatives(&discoveryResult, graph)

	c.log.Info("Phase 5: Building Final Endpoints for Output")
	return c.buildEndpoints(apiModel, discoveryResult, workingState, graph)
}

func (c *Compiler) buildEndpoints(apiModel *openapi.Model, result DiscoveryResult, state *WorkingState, graph *IdentityGraph) []*Endpoint {
	var endpoints []*Endpoint
	epMap := make(map[string]*Endpoint)

	for _, op := range apiModel.Operations {
		epKey := op.Method + ":" + op.Path
		epMap[epKey] = &Endpoint{
			Method: op.Method,
			Path:   op.Path,
		}
		endpoints = append(endpoints, epMap[epKey])
	}

	slotIdentity := func(cand IdentityCandidate) {
		epKey := cand.OriginMethod + ":" + cand.OriginPath
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
		epKey := cand.OriginMethod + ":" + cand.OriginPath
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
