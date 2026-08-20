package http

import (
	"strings"

	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

// Request is the "req" object passed to the .before(ctx, req) hook.
type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
}

func (r *Request) Reset() {
	r.URL = ""
	r.Method = ""
	r.Body = r.Body[:0]
	clear(r.Headers)
}

func (r *Request) SetBody(data any) {
	if data == nil {
		return
	}

	switch v := data.(type) {
	case string:
		r.Body = append(r.Body[:0], v...)
	case []byte:
		r.Body = append(r.Body[:0], v...)
	default:
		bytes, err := json.Marshal(v)
		if err == nil {
			r.Body = bytes
		}
	}
}

func (r *Request) SetHeader(key, val string) {
	r.Headers[key] = val
}

func (r *Request) SetPath(key, val string) {
	r.URL = strings.Replace(r.URL, "{"+key+"}", val, 1)
}

func (r *Request) SetQuery(key, val string) {
	var sb strings.Builder
	sb.WriteString(r.URL)
	if strings.Contains(r.URL, "?") {
		sb.WriteString("&")
	} else {
		sb.WriteString("?")
	}
	sb.WriteString(key)
	sb.WriteString("=")
	sb.WriteString(val)
	r.URL = sb.String()
}

func (r *Request) Get(path string) gjson.Result {
	if len(r.Body) == 0 {
		return gjson.Result{}
	}
	return gjson.GetBytes(r.Body, path)
}
func (r *Request) GetBody() string {
	return string(r.Body)
}
