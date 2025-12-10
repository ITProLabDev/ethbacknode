package endpoint

import (
	"github.com/valyala/fasthttp"
	"os"
)

const (
	MIME_TYPE_TEXT_HTML = "text/html"
)

type DevForm struct {
	FormPath string
}

func (d DevForm) ContentType() string {
	return MIME_TYPE_TEXT_HTML
}

func (d DevForm) StatusCode() int {
	return fasthttp.StatusOK
}

func (d DevForm) Body() string {
	html, err := os.ReadFile(d.FormPath)
	if err != nil {
		panic(err)
	}
	return string(html)
}
