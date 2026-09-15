package http

import (
	"bytes"
	"mime/multipart"
	"net/textproto"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"

	"github.com/budment/budment/internal/fastconv"
	"github.com/budment/budment/internal/template"
)

// Request represents the mutable HTTP request passed to the .before(req) pipeline hook.
type Request struct {
	Target  template.Expression
	Scope   template.ScopeProvider
	Method  string
	Headers map[string]string
	Body    []byte
}

// GetTarget evaluates and returns the rendered URL on-demand using the latest Scope.
func (r *Request) GetTarget() string {
	if r.Scope != nil && r.Target.Raw != "" {
		return r.Target.Render(r.Scope)
	}
	return r.Target.Raw
}

// Set mutates the outgoing payload and options (headers).
func (r *Request) Set(body any, options map[string]any) {
	if body != nil {
		switch v := body.(type) {
		case []byte:
			r.Body = v
		case string:
			r.Body = fastconv.StringToBytes(v)

		case interface{ Bytes() []byte }:
			r.Body = v.Bytes()

		case interface{ Export() any }:
			exported := v.Export()
			if b, ok := exported.([]byte); ok {
				r.Body = b
				goto ProcessOptions
			}
			if bb, ok := exported.(interface{ Bytes() []byte }); ok {
				r.Body = bb.Bytes()
				goto ProcessOptions
			}
			if fields, ok := exported.(map[string]any); ok && hasMultipartFile(fields) {
				r.buildMultipart(fields)
				goto ProcessOptions
			}
			if bytesVal, err := json.Marshal(exported); err == nil {
				r.Body = bytesVal
			}

		case []any:
			buf := make([]byte, len(v))
			for i, b := range v {
				switch num := b.(type) {
				case int64:
					buf[i] = byte(num)
				case int:
					buf[i] = byte(num)
				case float64:
					buf[i] = byte(num)
				}
			}
			r.Body = buf

		case map[string]any:
			if hasMultipartFile(v) {
				r.buildMultipart(v)
				goto ProcessOptions
			}
			if bytesVal, err := json.Marshal(v); err == nil {
				r.Body = bytesVal
			}

		default:
			if bytesVal, err := json.Marshal(v); err == nil {
				r.Body = bytesVal
			}
		}
	}

ProcessOptions:
	if options == nil {
		return
	}

	if rawHeaders, ok := options["headers"].(map[string]any); ok {
		if r.Headers == nil {
			r.Headers = make(map[string]string, len(rawHeaders))
		}
		for k, v := range rawHeaders {
			r.Headers[k] = fastconv.String(v)
		}
	}
}

// Json extracts a single value via GJSON or unmarshals the full payload if no selector is provided.
func (r *Request) Json(selector ...string) any {
	if len(r.Body) == 0 {
		return nil
	}

	if len(selector) > 0 && selector[0] != "" {
		res := gjson.GetBytes(r.Body, selector[0])
		if !res.Exists() {
			return nil
		}
		return res.Value()
	}

	var out any
	if err := json.Unmarshal(r.Body, &out); err != nil {
		return nil
	}
	return out
}

// ApplyMutation renders pre-compiled template headers and payloads into the request context.
func (r *Request) ApplyMutation(meta map[string]template.Expression, payload template.Expression, scope template.ScopeProvider) {
	if len(meta) > 0 {
		if r.Headers == nil {
			r.Headers = make(map[string]string, len(meta))
		}
		for k, expr := range meta {
			r.Headers[k] = expr.Render(scope)
		}
	}
	if payload.Raw != "" {
		r.Body = fastconv.StringToBytes(payload.Render(scope))
	}
}

// File creates a descriptor marking a payload as a multipart file upload.
// Exposed to the JavaScript runtime as req.file(data, [filename], [contentType]).
func (r *Request) File(data any, args ...string) map[string]any {
	filename := "upload.bin"
	contentType := "application/octet-stream"

	if len(args) > 0 && args[0] != "" {
		filename = args[0]
	}
	if len(args) > 1 && args[1] != "" {
		contentType = args[1]
	}

	return map[string]any{
		"__budment_file": true,
		"data":           data,
		"filename":       filename,
		"contentType":    contentType,
	}
}

// hasMultipartFile checks if any top-level key contains a file descriptor.
func hasMultipartFile(fields map[string]any) bool {
	for _, val := range fields {
		if m, ok := val.(map[string]any); ok {
			if isFile, _ := m["__budment_file"].(bool); isFile {
				return true
			}
		}
	}
	return false
}

// buildMultipart serializes form fields and file descriptors into a MIME multipart body.
func (r *Request) buildMultipart(fields map[string]any) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, val := range fields {
		if val == nil {
			continue
		}

		if fileMap, ok := val.(map[string]any); ok && fileMap["__budment_file"] == true {
			filename, _ := fileMap["filename"].(string)
			contentType, _ := fileMap["contentType"].(string)

			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", `form-data; name="`+key+`"; filename="`+filename+`"`)
			h.Set("Content-Type", contentType)

			part, err := writer.CreatePart(h)
			if err != nil {
				continue
			}

			var data []byte
			switch d := fileMap["data"].(type) {
			case []byte:
				data = d
			case string:
				data = fastconv.StringToBytes(d)
			case interface{ Bytes() []byte }:
				data = d.Bytes()
			case interface{ Export() any }:
				if raw, ok := d.Export().([]byte); ok {
					data = raw
				}
			}

			if len(data) > 0 {
				_, _ = part.Write(data)
			}
			continue
		}

		_ = writer.WriteField(key, fastconv.String(val))
	}

	_ = writer.Close()
	r.Body = buf.Bytes()

	if r.Headers == nil {
		r.Headers = make(map[string]string, 2)
	}
	r.Headers["Content-Type"] = writer.FormDataContentType()
}
