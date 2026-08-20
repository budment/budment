package http

import (
	"github.com/tidwall/gjson"
)

// Response is the "res" object passed to the .after(ctx, req, res) hook.
type Response struct {
	Status  int
	Headers map[string]string
	Error   string
	rawBody []byte
}

func NewResponse(status int, headers map[string]string, body []byte, errStr string) *Response {
	return &Response{
		Status:  status,
		Headers: headers,
		Error:   errStr,
		rawBody: body,
	}
}

func (r *Response) Get(path string) gjson.Result {
	if len(r.rawBody) == 0 {
		return gjson.Result{}
	}

	result := gjson.GetBytes(r.rawBody, path)
	return result
}

func (r *Response) GetBody() string {
	return string(r.rawBody)
}
