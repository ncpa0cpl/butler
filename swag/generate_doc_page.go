package swag

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
)

//go:embed doc_template.go.html
var tmpl string

//go:embed script.js
var js string

var funcMap = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
	"multiply": func(a, b int) int {
		return a * b
	},
	"tuple": func(els ...any) []any {
		return els
	},
	"flatten": func(els []EndpointData) []EndpointData {
		var res []EndpointData

		var addToRes func(endp EndpointData)
		addToRes = func(endp EndpointData) {
			if len(endp.Children) > 0 {
				for _, subEndp := range endp.Children {
					addToRes(subEndp)
				}
			} else {
				res = append(res, endp)
			}
		}

		for _, el := range els {
			addToRes(el)
		}

		return res
	},
	"atleasttwo": func(els ...bool) bool {
		count := 0
		for _, el := range els {
			if el {
				count++
			}
			if count >= 2 {
				return true
			}
		}
		return false
	},
}

func generateDocPage(endpoints []EndpointData) ([]byte, error) {
	t, err := template.New("doc_template.go.html").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return nil, err
	}

	var output bytes.Buffer
	err = t.Execute(&output, struct {
		PageTitle string
		Endpoints []EndpointData
		ScriptTag template.HTML
	}{
		PageTitle: "",
		Endpoints: endpoints,
		ScriptTag: template.HTML(fmt.Sprintf("<script>%s</script>", js)),
	})

	return output.Bytes(), err
}
