package endpoint

import (
	"strings"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/openapi"
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

func (d *Discoverer) Discover(apiModel *openapi.Model, state *WorkingState) DiscoveryResult {
	result := DiscoveryResult{}

	for _, op := range apiModel.Operations {
		currentResource := extractResource(op.Path)
		d.scanRequest(op, &result)

		for _, resp := range op.Responses {
			if !strings.HasPrefix(resp.StatusCode, "2") {
				continue
			}

			for _, header := range resp.Headers {
				if d.isIdentifier(header.Name) {
					ctx := SemanticContext{ResourceName: currentResource, IsCurrent: true}
					d.categorizeIdentity(header.Name, []string{"header", header.Name}, extractSchemaTypes(header.Schema), ctx, op, &result)
				}
			}

			for _, content := range resp.Contents {
				d.walkSchema(content.Schema, []string{"body"}, func(name string, path []string, schema *openapi.Schema) {
					if d.isIdentifier(name) || d.hasIdentifierType(schema) {
						semanticContext := d.resolveSemanticContext(name, path, currentResource)
						d.categorizeIdentity(name, path, extractSchemaTypes(schema), semanticContext, op, &result)
					}
				})
			}
		}
	}

	return result
}

func (d *Discoverer) scanRequest(op openapi.Operation, result *DiscoveryResult) {
	for _, param := range op.Parameters {
		if d.isIdentifier(param.Name) || d.hasIdentifierType(param.Schema) {
			path := []string{strings.ToLower(param.In), param.Name}
			cand := RelativeCandidate{
				Name:           param.Name,
				NodePath:       path,
				Types:          extractSchemaTypes(param.Schema),
				OriginProtocol: "rest",
				OriginMethod:   op.Method,
				OriginPath:     op.Path,
			}
			result.RelativeCandidates = append(result.RelativeCandidates, cand)
		}
	}

	if op.RequestBody != nil {
		for _, content := range op.RequestBody.Contents {
			d.walkSchema(content.Schema, []string{"body"}, func(name string, path []string, schema *openapi.Schema) {
				if d.isIdentifier(name) || d.hasIdentifierType(schema) {
					semanticName := d.getRelativeSemanticName(name, path)
					cand := RelativeCandidate{
						Name:           semanticName,
						NodePath:       path,
						Types:          extractSchemaTypes(schema),
						OriginProtocol: "rest",
						OriginMethod:   op.Method,
						OriginPath:     op.Path,
					}
					result.RelativeCandidates = append(result.RelativeCandidates, cand)
				}
			})
		}
	}
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

func (d *Discoverer) categorizeIdentity(name string, path []string, types []string, ctx SemanticContext, op openapi.Operation, result *DiscoveryResult) {
	candidate := IdentityCandidate{
		Name:           name,
		NodePath:       path,
		Types:          types,
		Context:        ctx,
		OriginProtocol: "rest",
		OriginMethod:   op.Method,
		OriginPath:     op.Path,
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

	for _, token := range d.identifierTokens {
		if lowerName == token {
			break
		}
		if strings.HasSuffix(lowerName, token) && len(name) > len(token) {
			tokenLen := len(token)
			charBefore := name[len(name)-tokenLen-1]

			if charBefore == '_' || charBefore == '-' || charBefore == '.' {
				owner = name[:len(name)-tokenLen-1]
				break
			}

			firstCharOfSuffix := name[len(name)-tokenLen]
			if firstCharOfSuffix >= 'A' && firstCharOfSuffix <= 'Z' {
				owner = name[:len(name)-tokenLen]
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

func (d *Discoverer) walkSchema(schema *openapi.Schema, currentPath []string, callback func(string, []string, *openapi.Schema)) {
	if schema == nil {
		return
	}
	visiting := make(map[*openapi.Schema]bool)
	d.doWalkSchema(schema, currentPath, visiting, callback)
}

func (d *Discoverer) doWalkSchema(schema *openapi.Schema, currentPath []string, visiting map[*openapi.Schema]bool, callback func(string, []string, *openapi.Schema)) {
	if schema == nil || visiting[schema] {
		return
	}
	visiting[schema] = true
	defer func() { visiting[schema] = false }()

	if schema.Items != nil {
		d.doWalkSchema(schema.Items, currentPath, visiting, callback)
	}

	for _, prop := range schema.Properties {
		newPath := append(append([]string(nil), currentPath...), prop.Name)
		callback(prop.Name, newPath, prop.Schema)
		d.doWalkSchema(prop.Schema, newPath, visiting, callback)
	}

	for _, s := range append(append(schema.AllOf, schema.AnyOf...), schema.OneOf...) {
		d.doWalkSchema(s, currentPath, visiting, callback)
	}
}

func (d *Discoverer) isIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}

	lowerName := strings.ToLower(name)
	for _, token := range d.identifierTokens {
		tokenLen := len(token)

		if lowerName == token {
			return true
		}

		if strings.HasSuffix(lowerName, token) && len(name) > tokenLen {
			charBefore := name[len(name)-tokenLen-1]
			if charBefore == '_' || charBefore == '-' || charBefore == '.' {
				return true
			}
			firstCharOfSuffix := name[len(name)-tokenLen]
			isCamelCase := (firstCharOfSuffix >= 'A' && firstCharOfSuffix <= 'Z') && (charBefore >= 'a' && charBefore <= 'z')
			if isCamelCase {
				return true
			}
		}

		if strings.HasPrefix(lowerName, token) && len(name) > tokenLen {
			charAfter := name[tokenLen]
			if charAfter == '_' || charAfter == '-' || charAfter == '.' {
				return true
			}
			isCamelCase := (charAfter >= 'A' && charAfter <= 'Z') && (name[tokenLen-1] >= 'a' && name[tokenLen-1] <= 'z')
			if isCamelCase {
				return true
			}
		}
	}
	return false
}

func (d *Discoverer) hasIdentifierType(schema *openapi.Schema) bool {
	if schema == nil {
		return false
	}
	f := strings.ToLower(schema.Format)
	return strings.Contains(f, "id") || strings.Contains(f, "uid")
}

func extractSchemaTypes(schema *openapi.Schema) []string {
	if schema == nil || len(schema.Types) == 0 {
		return nil
	}
	return schema.Types
}
