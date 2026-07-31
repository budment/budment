package endpoint

import (
	"strings"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/schema"
)

type Discoverer struct {
	identifierTokens []string
	wrapperNames     map[string]bool
}

func NewDiscoverer(cfg config.EndpointConfig) *Discoverer {
	wn := make(map[string]bool)
	for _, name := range cfg.WrapperNames {
		wn[strings.ToLower(name)] = true
	}

	return &Discoverer{
		identifierTokens: cfg.IdentifierTokens,
		wrapperNames:     wn,
	}
}

func (d *Discoverer) Discover(actions []schema.Action, state *WorkingState) DiscoveryResult {
	result := DiscoveryResult{}

	for _, action := range actions {
		currentResource := extractResource(action.Resource)

		// Scan Inputs for Relatives
		for _, field := range action.Inputs {
			if d.isIdentifier(field.Name) || d.hasIdentifierType(field.Format) {
				semanticName := d.getRelativeSemanticName(field.Name, field.Path)
				cand := RelativeCandidate{
					Name:           semanticName,
					NodePath:       field.Path,
					Types:          field.Types,
					OriginProtocol: action.Protocol,
					OriginMethod:   action.Method,
					OriginPath:     action.Resource,
				}
				result.RelativeCandidates = append(result.RelativeCandidates, cand)
			}
		}

		// Scan Outputs for Identities
		for _, field := range action.Outputs {
			if d.isIdentifier(field.Name) || d.hasIdentifierType(field.Format) {
				ctx := d.resolveSemanticContext(field.Name, field.Path, currentResource)
				if field.IsHeader {
					ctx.IsCurrent = true // Headers natively belong to the resource payload
				}
				d.categorizeIdentity(field.Name, field.Path, field.Types, ctx, action, &result)
			}
		}
	}

	return result
}

func (d *Discoverer) getRelativeSemanticName(name string, path []string) string {
	lowerName := strings.ToLower(name)
	isIdentifier := false
	for _, token := range d.identifierTokens {
		if lowerName == token {
			isIdentifier = true
			break
		}
	}

	if !isIdentifier {
		return name
	}

	if len(path) > 1 {
		for i := len(path) - 2; i >= 0; i-- {
			parent := path[i]
			if !d.wrapperNames[strings.ToLower(parent)] {
				return parent + strings.Title(name)
			}
		}
	}
	return name
}

func (d *Discoverer) categorizeIdentity(name string, path []string, types []string, ctx SemanticContext, action schema.Action, result *DiscoveryResult) {
	candidate := IdentityCandidate{
		Name:           name,
		NodePath:       path,
		Types:          types,
		Context:        ctx,
		OriginProtocol: action.Protocol,
		OriginMethod:   action.Method,
		OriginPath:     action.Resource,
	}

	if ctx.IsCurrent {
		candidate.Category = CategoryRoot
		result.RootCandidates = append(result.RootCandidates, candidate)
	} else {
		candidate.Category = CategoryBranch
		candidate.TargetResource = ctx.ResourceName
		result.BranchCandidates = append(result.BranchCandidates, candidate)
	}
}

func (d *Discoverer) resolveSemanticContext(name string, path []string, currentResource string) SemanticContext {
	owner := ""
	lowerName := strings.ToLower(name)
	nameRunes := []rune(name)
	nLen := len(nameRunes)

	for _, token := range d.identifierTokens {
		if lowerName == token {
			break
		}

		tRunes := []rune(token)
		tLen := len(tRunes)

		if strings.HasSuffix(lowerName, token) && nLen > tLen {
			charBefore := nameRunes[nLen-tLen-1]
			if charBefore == '_' || charBefore == '-' || charBefore == '.' {
				owner = string(nameRunes[:nLen-tLen-1])
				break
			}
			firstCharOfSuffix := nameRunes[nLen-tLen]
			if firstCharOfSuffix >= 'A' && firstCharOfSuffix <= 'Z' {
				owner = string(nameRunes[:nLen-tLen])
				break
			}
		}
	}

	if owner == "" && len(path) > 1 {
		for i := len(path) - 2; i >= 0; i-- {
			parent := strings.ToLower(path[i])
			if !d.wrapperNames[parent] {
				owner = parent
				break
			}
		}
	}

	if owner == "" {
		owner = currentResource
	}

	ctx := SemanticContext{ResourceName: owner, IsCurrent: false}
	if strings.EqualFold(owner, currentResource) {
		ctx.IsCurrent = true
	}
	return ctx
}

func (d *Discoverer) isIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}
	lowerName := strings.ToLower(name)
	nameRunes := []rune(name)
	nLen := len(nameRunes)

	for _, token := range d.identifierTokens {
		tRunes := []rune(token)
		tLen := len(tRunes)

		if lowerName == token {
			return true
		}

		if strings.HasSuffix(lowerName, token) && nLen > tLen {
			charBefore := nameRunes[nLen-tLen-1]
			if charBefore == '_' || charBefore == '-' || charBefore == '.' {
				return true
			}
			firstCharOfSuffix := nameRunes[nLen-tLen]
			isCamelCase := (firstCharOfSuffix >= 'A' && firstCharOfSuffix <= 'Z') && (charBefore >= 'a' && charBefore <= 'z')
			if isCamelCase {
				return true
			}
		}

		if strings.HasPrefix(lowerName, token) && nLen > tLen {
			charAfter := nameRunes[tLen]
			if charAfter == '_' || charAfter == '-' || charAfter == '.' {
				return true
			}
			isCamelCase := (charAfter >= 'A' && charAfter <= 'Z') && (nameRunes[tLen-1] >= 'a' && nameRunes[tLen-1] <= 'z')
			if isCamelCase {
				return true
			}
		}
	}
	return false
}

func (d *Discoverer) hasIdentifierType(format string) bool {
	f := strings.ToLower(format)
	return strings.Contains(f, "id") || strings.Contains(f, "uid")
}
