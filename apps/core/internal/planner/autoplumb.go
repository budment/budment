package planner

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/vunas/blaster/internal/discovery/endpoint"
)

type EndpointDef struct {
	AbsAddress string
	Produces   []IdentityDef
	Consumes   []RelativeDef
}
type IdentityDef struct{ TargetNode, Path, StackID string }
type RelativeDef struct{ TargetNode, InjectPath, StackID string }

type AutoPlumber struct {
	EndpointRegistry map[string]*EndpointDef
	SetupProduces    map[string]bool
	RefCounts        map[string]int
}

func normalizeAddress(targetID string) string {
	parts := strings.Split(targetID, ":")
	if len(parts) == 3 {
		return fmt.Sprintf("rest:%s:%s:%s", strings.ToUpper(parts[0]), parts[1], parts[2])
	} else if len(parts) >= 4 {
		return fmt.Sprintf("%s:%s:%s:%s", parts[0], strings.ToUpper(parts[1]), parts[2], parts[3])
	}
	return targetID
}

func NewAutoPlumber(rawEndpoints []*endpoint.Endpoint) *AutoPlumber {
	registry := make(map[string]*EndpointDef)

	for _, ep := range rawEndpoints {
		cleanPath := strings.TrimSuffix(ep.Path, "/")
		absAddress := fmt.Sprintf("%s:%s:%s", ep.Protocol, strings.ToUpper(ep.Method), cleanPath)
		def := &EndpointDef{AbsAddress: absAddress}

		// Resolve producer identities.
		for _, id := range ep.Identities {
			if id.Status == endpoint.StatusResolved {
				stackID := ""

				// Root identities own their own stack; branches reuse the target stack.
				if id.TargetID == "identity" || id.TargetID == "" {
					stackID = fmt.Sprintf("%s:%s", absAddress, id.Name)
				} else {
					stackID = normalizeAddress(id.TargetID)
				}

				def.Produces = append(def.Produces, IdentityDef{
					TargetNode: id.Name,
					Path:       strings.Join(id.NodePath, "."),
					StackID:    stackID,
				})
			}
		}

		// Resolve consumer relatives.
		for _, rel := range ep.Relatives {
			if rel.Status == endpoint.StatusResolved && rel.TargetID != "" {
				// Resolve the referenced stack.
				stackID := normalizeAddress(rel.TargetID)

				def.Consumes = append(def.Consumes, RelativeDef{
					TargetNode: rel.TargetID,
					InjectPath: strings.Join(rel.NodePath, "."),
					StackID:    stackID,
				})
			}
		}
		registry[absAddress] = def
	}

	return &AutoPlumber{
		EndpointRegistry: registry,
		SetupProduces:    make(map[string]bool),
		RefCounts:        make(map[string]int),
	}
}

func (p *AutoPlumber) Plumb(graph *Graph) error {
	if len(p.EndpointRegistry) == 0 {
		return nil
	}

	// collect references and initialize extraction slots.
	p.walkPass1(graph.Setup, true)
	p.walkPass1(graph.Execution, false)

	// remove unreferenced extractions.
	p.walkPass2(graph.Setup)
	p.walkPass2(graph.Execution)

	return nil
}

func (p *AutoPlumber) walkPass1(nodes []ExecutableNode, isSetup bool) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *HttpNode:
			apiPath := strings.TrimSuffix(n.URL, "/")
			if parsedURL, err := url.Parse(apiPath); err == nil && parsedURL.Path != "" {
				apiPath = parsedURL.Path
			}
			absAddress := fmt.Sprintf("rest:%s:%s", strings.ToUpper(n.Method), apiPath)

			def, exists := p.EndpointRegistry[absAddress]
			if !exists {
				continue
			}

			// Consumer: +1 to the counter
			for _, cons := range def.Consumes {
				isFromDist := p.SetupProduces[cons.StackID] && !isSetup
				n.Injects = append(n.Injects, InjectInstruction{
					StackID:          cons.StackID,
					Target:           cons.InjectPath,
					IsFromDistribute: isFromDist,
				})
				// Increment the counter for this StackID
				p.RefCounts[cons.StackID]++
			}

			// Register producer extractions.
			for _, prod := range def.Produces {
				if isSetup {
					p.SetupProduces[prod.StackID] = true
				}
				n.Extracts = append(n.Extracts, ExtractInstruction{
					StackID:      prod.StackID,
					Path:         prod.Path,
					IsDistribute: isSetup,
				})
			}

		case *BranchNode:
			p.walkPass1(n.TruePath, isSetup)
			p.walkPass1(n.FalsePath, isSetup)
		case *LoopNode:
			p.walkPass1(n.Logic, isSetup)
		}
	}
}

func (p *AutoPlumber) walkPass2(nodes []ExecutableNode) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *HttpNode:
			// Keep only extractions with active consumers.
			var activeExtracts []ExtractInstruction
			for _, ext := range n.Extracts {
				refCount := p.RefCounts[ext.StackID]
				if refCount > 0 {
					ext.TotalRef = refCount
					activeExtracts = append(activeExtracts, ext)
				}
			}
			n.Extracts = activeExtracts

		case *BranchNode:
			p.walkPass2(n.TruePath)
			p.walkPass2(n.FalsePath)
		case *LoopNode:
			p.walkPass2(n.Logic)
		}
	}
}
