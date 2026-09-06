package http

import (
	"net/url"
	"strings"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"

	"github.com/vunas/blaster/internal/fastconv"
	"github.com/vunas/blaster/internal/template"
)

// Request is the "req" object passed to the .before(req) hook.
type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
}

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

	if pathVal, ok := options["path"].(string); ok && pathVal != "" {
		if strings.HasPrefix(pathVal, "http://") || strings.HasPrefix(pathVal, "https://") {
			r.URL = pathVal
		} else if u, err := url.Parse(r.URL); err == nil {
			if strings.HasPrefix(pathVal, "/") {
				u.Path = pathVal
			} else {
				u.Path = strings.TrimSuffix(u.Path, "/") + "/" + pathVal
			}
			r.URL = u.String()
		}
	}

	if rawParams, ok := options["params"].(map[string]any); ok {
		if u, err := url.Parse(r.URL); err == nil {
			q := u.Query()
			for k, v := range rawParams {
				q.Set(k, fastconv.String(v))
			}
			u.RawQuery = q.Encode()
			r.URL = u.String()
		}
	}
}

// Quickly extract a single node using GJSON or unmarshal the entire body.
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
