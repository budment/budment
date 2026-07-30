package endpoint

import (
	"strings"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/similarity"
)

type Resolver struct {
	MinimumScore     float64
	SafetyMargin     float64
	matcher          similarity.Matcher
	learnedMap       map[string]string
	identifierTokens []string
}

func NewResolver(cfg config.EndpointConfig) *Resolver {
	return &Resolver{
		MinimumScore:     cfg.MinimumScore,
		SafetyMargin:     cfg.SafetyMargin,
		matcher:          similarity.NewJaroWinkler(cfg.JaroBoostThreshold, cfg.JaroPrefixSize),
		learnedMap:       make(map[string]string),
		identifierTokens: cfg.IdentifierTokens,
	}
}

func (r *Resolver) BuildGraphAndResolveBranches(result *DiscoveryResult, state *WorkingState) *IdentityGraph {
	graph := NewIdentityGraph()
	canonicalMap := make(map[string]*RootNode)

	for _, id := range state.LockedIdentities {
		if id.Status == StatusResolved && id.TargetID != "" {
			r.learnedMap[strings.ToLower(id.Name)] = id.TargetID
		}
	}
	for _, rel := range state.LockedRelatives {
		if rel.Status == StatusResolved && rel.TargetID != "" {
			r.learnedMap[strings.ToLower(rel.Name)] = rel.TargetID
		}
	}

	for epKey, ident := range state.LockedIdentities {
		parts := strings.Split(epKey, "|")
		if len(parts) == 2 {
			epRouteParts := strings.SplitN(parts[0], ":", 3)
			if len(epRouteParts) == 3 {
				protocol := epRouteParts[0]
				method := epRouteParts[1]
				path := epRouteParts[2]

				if ident.Status == StatusResolved && ident.TargetID == "" {
					owner := r.extractSemanticToken(ident.Name)
					if owner == ident.Name {
						owner = extractResource(path)
					}

					node := &RootNode{
						Protocol:      protocol,
						Method:        method,
						Resource:      path,
						SemanticOwner: owner,
						Name:          ident.Name,
						Types:         ident.Types,
					}

					key := strings.ToLower(node.Protocol + ":" + node.Resource + ":" + node.Name)
					canonicalMap[key] = node
					graph.Roots[node.GlobalID()] = node
				}
			}
		}
	}

	for i := range result.RootCandidates {
		root := &result.RootCandidates[i]
		key := strings.ToLower(root.OriginProtocol + ":" + root.OriginPath + ":" + root.Name)

		if existing, exists := canonicalMap[key]; exists {
			root.Category = CategoryBranch
			root.TargetID = existing.GlobalID()
		} else {
			node := &RootNode{
				Protocol:      root.OriginProtocol,
				Method:        root.OriginMethod,
				Resource:      root.OriginPath,
				SemanticOwner: extractResource(root.OriginPath),
				Name:          root.Name,
				Types:         root.Types,
			}
			canonicalMap[key] = node
			graph.Roots[node.GlobalID()] = node
		}
	}

	for i := range result.BranchCandidates {
		branch := &result.BranchCandidates[i]
		branchNameLower := strings.ToLower(branch.Name)

		if target, ok := r.learnedMap[branchNameLower]; ok {
			if _, exists := graph.Roots[target]; exists {
				branch.TargetID = target
				continue
			}
		}

		bestScore := -1.0
		secondBestScore := -1.0
		var targetRootID string

		for id, rootNode := range graph.Roots {
			if !r.isTypeCompatible(branch.Types, rootNode.Types) {
				continue
			}

			score := r.calculateDeterministicScore(branch.Name, branch.OriginPath, rootNode.SemanticOwner, rootNode.Name)

			if score > bestScore {
				secondBestScore = bestScore
				bestScore = score
				targetRootID = id
			} else if score > secondBestScore {
				secondBestScore = score
			}
		}

		if bestScore >= r.MinimumScore && (bestScore-secondBestScore) >= r.SafetyMargin {
			branch.TargetID = targetRootID
		}
	}

	return graph
}

func (r *Resolver) ResolveRelatives(result *DiscoveryResult, graph *IdentityGraph) {
	for i := range result.RelativeCandidates {
		rel := &result.RelativeCandidates[i]
		relNameLower := strings.ToLower(rel.Name)

		if target, ok := r.learnedMap[relNameLower]; ok {
			if _, exists := graph.Roots[target]; exists {
				rel.TargetID = target
				continue
			}
		}

		bestScore := -1.0
		secondBestScore := -1.0
		var bestRootID string

		for id, root := range graph.Roots {
			if !r.isTypeCompatible(rel.Types, root.Types) {
				continue
			}

			score := r.calculateDeterministicScore(rel.Name, rel.OriginPath, root.SemanticOwner, root.Name)

			if score > bestScore {
				secondBestScore = bestScore
				bestScore = score
				bestRootID = id
			} else if score > secondBestScore {
				secondBestScore = score
			}
		}

		if bestScore >= r.MinimumScore && (bestScore-secondBestScore) >= r.SafetyMargin {
			rel.TargetID = bestRootID
		}
	}
}

func (r *Resolver) calculateDeterministicScore(relName, relRoute, rootSemanticOwner, rootName string) float64 {
	relLower := strings.ToLower(relName)
	rootNameLower := strings.ToLower(rootName)

	if relLower == rootNameLower {
		relOwner := extractResource(relRoute)
		if strings.EqualFold(relOwner, rootSemanticOwner) {
			return 100.0
		}
	}

	cleanRelBase := r.extractSemanticToken(relName)
	cleanRootBase := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(rootSemanticOwner), "-", ""), "_", "")

	if len(cleanRelBase) >= 2 {
		similarityScore := r.matcher.Calculate(cleanRelBase, cleanRootBase) * 100.0
		return similarityScore
	}

	return 0.0
}

func (r *Resolver) extractSemanticToken(originalName string) string {
	lower := strings.ToLower(originalName)

	base := lower
	for _, t := range r.identifierTokens {
		if strings.HasSuffix(base, t) && len(base) > len(t) {
			base = base[:len(base)-len(t)]
			break
		}
	}
	base = strings.TrimRight(base, "_-.")

	lastSep := -1
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '_' || base[i] == '-' || base[i] == '.' {
			lastSep = i
			break
		}
	}

	if lastSep >= 0 && lastSep < len(base)-1 {
		return base[lastSep+1:]
	}

	lastUpper := -1
	for i := 1; i < len(base); i++ {
		if originalName[i] >= 'A' && originalName[i] <= 'Z' {
			lastUpper = i
		}
	}

	if lastUpper > 0 && lastUpper < len(base)-1 {
		return strings.ToLower(base[lastUpper:])
	}

	return base
}

func (r *Resolver) isTypeCompatible(consumerTypes, producerTypes []string) bool {
	if len(consumerTypes) == 0 || len(producerTypes) == 0 {
		return true
	}
	pMap := make(map[string]bool)
	for _, pt := range producerTypes {
		pMap[pt] = true
	}
	for _, ct := range consumerTypes {
		if pMap[ct] {
			return true
		}
		if (ct == "string" && pMap["integer"]) || (ct == "integer" && pMap["string"]) {
			return true
		}
	}
	return false
}

func extractResource(path string) string {
	cleanPath := strings.Trim(path, "/")
	for {
		if cleanPath == "" {
			return "root"
		}

		idx := strings.LastIndexByte(cleanPath, '/')
		var part string
		if idx == -1 {
			part = cleanPath
			cleanPath = ""
		} else {
			part = cleanPath[idx+1:]
			cleanPath = cleanPath[:idx]
		}

		if part != "" && !strings.HasPrefix(part, "{") {
			return part
		}
	}
}
