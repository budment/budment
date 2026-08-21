package http

import (
	"github.com/tidwall/gjson"
)

// Response is the "res" object passed to the .after(ctx, req, res) hook.
type Response struct {
	Status  int
	Headers map[string]string
	Error   string
	Body    []byte
}

func NewResponse(status int, headers map[string]string, body []byte, errStr string) *Response {
	return &Response{
		Status:  status,
		Headers: headers,
		Error:   errStr,
		Body:    body,
	}
}

func (r *Response) Get(path string) gjson.Result {
	if len(r.Body) == 0 {
		return gjson.Result{}
	}

	result := gjson.GetBytes(r.Body, path)
	return result
}

func (r *Response) GetBody() string {
	return string(r.Body)
}
