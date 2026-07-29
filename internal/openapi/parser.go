package openapi

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Parser converts raw OpenAPI 3.x bytes into the package internal model.
// It guarantees deterministic ordering and safe schema traversal.
type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(input []byte) (*Model, error) {
	if len(strings.TrimSpace(string(input))) == 0 {
		return nil, ErrEmptyInput
	}

	doc, err := p.loadDocument(input)
	if err != nil {
		return nil, err
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate OpenAPI document: %w", err)
	}

	return p.buildModel(doc), nil
}

func (p *Parser) loadDocument(input []byte) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false // Expecting a bundled spec for deterministic builds

	doc, err := loader.LoadFromData(input)
	if err != nil {
		return nil, fmt.Errorf("parse OpenAPI document: %w", err)
	}

	if !strings.HasPrefix(doc.OpenAPI, "3.") {
		return nil, fmt.Errorf("unsupported OpenAPI version %q: only 3.x is supported", doc.OpenAPI)
	}

	return doc, nil
}

func (p *Parser) buildModel(doc *openapi3.T) *Model {
	model := &Model{
		OpenAPIVersion: doc.OpenAPI,
		Info: Info{
			Title:       doc.Info.Title,
			Description: doc.Info.Description,
			Version:     doc.Info.Version,
		},
		Servers: toServers(doc.Servers),
	}
	model.Operations = toOperations(doc.Paths)
	return model
}

func toServers(servers openapi3.Servers) []Server {
	out := make([]Server, 0, len(servers))
	for _, server := range servers {
		if server != nil {
			out = append(out, Server{
				URL:         server.URL,
				Description: server.Description,
			})
		}
	}
	return out
}

// toOperations flattens paths and strictly enforces Design Doc Sorting Rules.
func toOperations(paths *openapi3.Paths) []Operation {
	if paths == nil || len(paths.Map()) == 0 {
		return nil
	}

	pathKeys := make([]string, 0, len(paths.Map()))
	for path := range paths.Map() {
		pathKeys = append(pathKeys, path)
	}

	// RULE: Shallower resources are processed before deeper resources.
	// Fallback to alphabetical for deterministic tie-breaking.
	sort.Slice(pathKeys, func(i, j int) bool {
		depthI := strings.Count(pathKeys[i], "/")
		depthJ := strings.Count(pathKeys[j], "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		return pathKeys[i] < pathKeys[j]
	})

	var out []Operation
	for _, path := range pathKeys {
		pathItem := paths.Map()[path]
		if pathItem == nil {
			continue
		}

		methodOps := collectOperations(pathItem)

		// RULE: Method priority: GET -> POST -> PUT -> PATCH -> DELETE
		sort.Slice(methodOps, func(i, j int) bool {
			return methodPriority(methodOps[i].method) < methodPriority(methodOps[j].method)
		})

		for _, methodOp := range methodOps {
			out = append(out, Operation{
				Path:        path,
				Method:      strings.ToUpper(methodOp.method),
				OperationID: methodOp.operation.OperationID,
				Summary:     methodOp.operation.Summary,
				Description: methodOp.operation.Description,
				Tags:        slices.Clone(methodOp.operation.Tags),
				Deprecated:  methodOp.operation.Deprecated,
				Parameters:  toParameters(pathItem.Parameters, methodOp.operation.Parameters),
				RequestBody: toRequestBody(methodOp.operation.RequestBody),
				Responses:   toResponses(methodOp.operation.Responses),
				Security:    toSecurityRequirements(methodOp.operation.Security),
			})
		}
	}
	return out
}

func methodPriority(method string) int {
	switch strings.ToLower(method) {
	case "get":
		return 1
	case "post":
		return 2
	case "put":
		return 3
	case "patch":
		return 4
	case "delete":
		return 5
	default:
		return 99 // Unconventional methods go last
	}
}

type methodOperation struct {
	method    string
	operation *openapi3.Operation
}

func collectOperations(item *openapi3.PathItem) []methodOperation {
	methods := []struct {
		m  string
		op *openapi3.Operation
	}{
		{"get", item.Get}, {"post", item.Post}, {"put", item.Put},
		{"patch", item.Patch}, {"delete", item.Delete},
		{"options", item.Options}, {"head", item.Head}, {"trace", item.Trace},
	}

	var out []methodOperation
	for _, m := range methods {
		if m.op != nil {
			out = append(out, methodOperation{method: m.m, operation: m.op})
		}
	}
	return out
}

func toParameters(pathParams, opParams openapi3.Parameters) []Parameter {
	merged := make(map[string]Parameter)
	converter := newSchemaConverter()

	addParams := func(params openapi3.Parameters) {
		for _, ref := range params {
			if ref != nil && ref.Value != nil {
				key := fmt.Sprintf("%s\x00%s", ref.Value.In, ref.Value.Name)
				merged[key] = Parameter{
					Name:        ref.Value.Name,
					In:          ref.Value.In,
					Required:    ref.Value.Required,
					Description: ref.Value.Description,
					Deprecated:  ref.Value.Deprecated,
					Schema:      converter.convert(ref.Value.Schema),
				}
			}
		}
	}

	addParams(pathParams)
	addParams(opParams)

	out := make([]Parameter, 0, len(merged))
	for _, p := range merged {
		out = append(out, p)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].In != out[j].In {
			return out[i].In < out[j].In
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func toRequestBody(ref *openapi3.RequestBodyRef) *RequestBody {
	if ref == nil || ref.Value == nil {
		return nil
	}
	return &RequestBody{
		Description: ref.Value.Description,
		Required:    ref.Value.Required,
		Contents:    toContents(ref.Value.Content),
	}
}

func toResponses(responses *openapi3.Responses) []Response {
	if responses == nil || len(responses.Map()) == 0 {
		return nil
	}

	codes := make([]string, 0, len(responses.Map()))
	for code := range responses.Map() {
		codes = append(codes, code)
	}

	sort.Slice(codes, func(i, j int) bool {
		return compareStatusCode(codes[i], codes[j]) < 0
	})

	var out []Response
	for _, code := range codes {
		val := responses.Map()[code]
		if val == nil || val.Value == nil {
			continue
		}
		desc := ""
		if val.Value.Description != nil {
			desc = *val.Value.Description
		}
		out = append(out, Response{
			StatusCode:  code,
			Description: desc,
			Headers:     toHeaders(val.Value.Headers),
			Contents:    toContents(val.Value.Content),
		})
	}
	return out
}

func toHeaders(headers openapi3.Headers) []Header {
	if len(headers) == 0 {
		return nil
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)

	converter := newSchemaConverter()
	var out []Header
	for _, name := range names {
		val := headers[name]
		if val != nil && val.Value != nil {
			out = append(out, Header{
				Name:        name,
				Description: val.Value.Description,
				Required:    val.Value.Required,
				Deprecated:  val.Value.Deprecated,
				Schema:      converter.convert(val.Value.Parameter.Schema),
			})
		}
	}
	return out
}

func toContents(content openapi3.Content) []Content {
	if len(content) == 0 {
		return nil
	}
	mediaTypes := make([]string, 0, len(content))
	for mt := range content {
		mediaTypes = append(mediaTypes, mt)
	}
	sort.Strings(mediaTypes)

	converter := newSchemaConverter()
	var out []Content
	for _, mt := range mediaTypes {
		if content[mt] != nil {
			out = append(out, Content{
				MediaType: mt,
				Schema:    converter.convert(content[mt].Schema),
			})
		}
	}
	return out
}

func toSecurityRequirements(reqs *openapi3.SecurityRequirements) []SecurityRequirement {
	if reqs == nil {
		return nil
	}
	var out []SecurityRequirement
	for _, req := range *reqs {
		names := make([]string, 0, len(req))
		for name := range req {
			names = append(names, name)
		}
		sort.Strings(names)

		var schemes []SecuritySchemeRequirement
		for _, name := range names {
			scopes := slices.Clone(req[name])
			sort.Strings(scopes)
			schemes = append(schemes, SecuritySchemeRequirement{
				Name:   name,
				Scopes: scopes,
			})
		}
		out = append(out, SecurityRequirement{Schemes: schemes})
	}
	return out
}

func compareStatusCode(left, right string) int {
	if left == "default" && right != "default" {
		return 1
	}
	if right == "default" && left != "default" {
		return -1
	}

	l, lErr := strconv.Atoi(left)
	r, rErr := strconv.Atoi(right)
	if lErr == nil && rErr == nil {
		return l - r
	}
	return strings.Compare(left, right)
}

// schemaConverter handles infinite recursive references safely
type schemaConverter struct {
	seen map[*openapi3.SchemaRef]*Schema
}

func newSchemaConverter() *schemaConverter {
	return &schemaConverter{
		seen: make(map[*openapi3.SchemaRef]*Schema),
	}
}

func (c *schemaConverter) convert(ref *openapi3.SchemaRef) *Schema {
	if ref == nil {
		return nil
	}

	// Cycle break
	if existing, ok := c.seen[ref]; ok {
		return existing
	}

	schema := &Schema{Ref: ref.Ref}
	c.seen[ref] = schema // Register pointer immediately before recursion

	val := ref.Value
	if val == nil {
		return schema
	}

	if val.Type != nil {
		schema.Types = slices.Clone(*val.Type)
		sort.Strings(schema.Types)
	}
	schema.Format = val.Format
	schema.Description = val.Description
	schema.Nullable = val.Nullable
	schema.Required = slices.Clone(val.Required)
	sort.Strings(schema.Required)

	for _, enum := range val.Enum {
		schema.Enum = append(schema.Enum, fmt.Sprint(enum))
	}
	sort.Strings(schema.Enum)

	var propNames []string
	for k := range val.Properties {
		propNames = append(propNames, k)
	}
	sort.Strings(propNames)

	for _, name := range propNames {
		schema.Properties = append(schema.Properties, SchemaProperty{
			Name:   name,
			Schema: c.convert(val.Properties[name]),
		})
	}

	schema.Items = c.convert(val.Items)
	schema.OneOf = c.convertSlice(val.OneOf)
	schema.AnyOf = c.convertSlice(val.AnyOf)
	schema.AllOf = c.convertSlice(val.AllOf)

	if val.AdditionalProperties.Schema != nil {
		schema.AdditionalProperties = c.convert(val.AdditionalProperties.Schema)
	}

	return schema
}

func (c *schemaConverter) convertSlice(in openapi3.SchemaRefs) []*Schema {
	var out []*Schema
	for _, ref := range in {
		out = append(out, c.convert(ref))
	}
	return out
}
