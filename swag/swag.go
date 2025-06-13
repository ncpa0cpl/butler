package swag

import (
	"github.com/labstack/echo/v4"
)

type EndpointData struct {
	Name        string
	Description string
	Children    []EndpointData
	Method      string
	Path        string
	IsGroup     bool
	ParamsT     TypeStructure
	BodyT       TypeStructure
	ResponseT   TypeStructure
}

func CreateApiDocumentation(path string, endpoints []EndpointData, e *echo.Echo) {
	html, err := generateDocPage(endpoints)

	if err != nil {
		e.Logger.Error("failed to generate a api doc page: ", err)
		return
	}

	e.GET(path, func(ctx echo.Context) error {
		return ctx.HTMLBlob(200, html)
	})
}
