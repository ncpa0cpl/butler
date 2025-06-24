package swag

import (
	"github.com/labstack/echo/v4"
)

type CacheOptions struct {
	DisableAutoResponseSkipping bool
	DisableETagGeneration       bool
	MaxAge                      string
	SMaxAge                     string
	StaleWhileRevalidate        string
	StaleIfError                string
	Immutable                   bool
	NoStore                     bool
	NoCache                     bool
	MustRevalidate              bool
	Private                     bool
	ProxyRevalidate             bool
	MustUnderstand              bool
	NoTransform                 bool
	ExampleHeader               string
}

type EndpointData struct {
	Uid             string
	Name            string
	Description     string
	Children        []EndpointData
	Method          string
	Path            string
	IsGroup         bool
	RespContentType string
	CacheOptions    *CacheOptions
	Encoding        string
	ParamsT         TypeStructure
	BodyT           TypeStructure
	ResponseT       TypeStructure
}

func CreateApiDocumentation(path string, endpoints []EndpointData, e *echo.Echo, m ...echo.MiddlewareFunc) {
	html, err := generateDocPage(endpoints)

	if err != nil {
		e.Logger.Error("failed to generate a api doc page: ", err)
		return
	}

	e.GET(path, func(ctx echo.Context) error {
		return ctx.HTMLBlob(200, html)
	}, m...)
}
