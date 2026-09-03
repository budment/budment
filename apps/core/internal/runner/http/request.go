package http

import (
	"net/url"
	"strings"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"

	"github.com/vunas/blaster/internal/fastconv"
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
		default:
			if bytesVal, err := json.Marshal(v); err == nil {
				r.Body = bytesVal
			}
		}
	}

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
func (r *Request) JSON(selector ...string) any {
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
