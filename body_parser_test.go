package butler_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	f "github.com/ncpa0cpl/butler"
	"github.com/stretchr/testify/assert"
)

type requireAllBody struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type requireAllNestedBody struct {
	Name    string            `json:"name"`
	Address requireAllAddress `json:"address"`
}

type requireAllAddress struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type taggedRequiredBody struct {
	Required string `json:"required" required:"true"`
	Optional string `json:"optional"`
}

type formBody struct {
	Name     string `form:"name"`
	Required string `form:"req" required:"true"`
}

func startServer(t *testing.T, endpoint f.EndpointInterface) {
	t.Helper()
	server := f.CreateServer()
	server.Port = 8080
	server.Add(endpoint)
	go server.Listen()
	t.Cleanup(server.Close)
	// wait for the server to accept connections
	waitUntil(func() bool {
		resp, err := http.Get("http://localhost:8080/")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return true
	})
}

func TestRequireAllBodyField(t *testing.T) {
	assert := assert.New(t)

	endpoint := &f.Endpoint[f.NoParams, requireAllBody]{
		Method:              "POST",
		Path:                "/required-all",
		RequireAllBodyField: true,
		Handler: func(request *f.Request, params f.NoParams, body *requireAllBody) *f.Response {
			return f.Respond.Ok().JSON(body)
		},
	}
	startServer(t, endpoint)

	body, resp := request("POST", "http://localhost:8080/required-all", &requireAllBody{Name: "a", Count: 1})
	assert.Equal(200, resp.StatusCode)
	assert.Equal(`{"name":"a","count":1}`, string(body))

	// a field set to its zero value is still present
	_, resp = request("POST", "http://localhost:8080/required-all", &requireAllBody{Name: "a", Count: 0})
	assert.Equal(200, resp.StatusCode)

	payload := strings.NewReader(`{"name":"a"}`)
	resp2 := rawRequest("POST", "http://localhost:8080/required-all", payload, "application/json")
	assert.Equal(400, resp2.StatusCode)

	// explicitly null field counts as missing
	payload = strings.NewReader(`{"name":"a","count":null}`)
	resp2 = rawRequest("POST", "http://localhost:8080/required-all", payload, "application/json")
	assert.Equal(400, resp2.StatusCode)

	resp = rawRequest("POST", "http://localhost:8080/required-all", nil, "application/json")
	assert.Equal(400, resp.StatusCode)
}

func TestRequireAllBodyFieldWithNestedStructs(t *testing.T) {
	assert := assert.New(t)

	endpoint := &f.Endpoint[f.NoParams, requireAllNestedBody]{
		Method:              "POST",
		Path:                "/required-nested",
		RequireAllBodyField: true,
		Handler: func(request *f.Request, params f.NoParams, body *requireAllNestedBody) *f.Response {
			return f.Respond.Ok().JSON(body)
		},
	}
	startServer(t, endpoint)

	_, resp := request("POST", "http://localhost:8080/required-nested",
		&requireAllNestedBody{Name: "a", Address: requireAllAddress{Street: "s", City: "c"}})
	assert.Equal(200, resp.StatusCode)

	payload := strings.NewReader(`{"name":"a","address":{"street":"s"}}`)
	resp2 := rawRequest("POST", "http://localhost:8080/required-nested", payload, "application/json")
	assert.Equal(400, resp2.StatusCode)

	payload = strings.NewReader(`{"name":"a"}`)
	resp2 = rawRequest("POST", "http://localhost:8080/required-nested", payload, "application/json")
	assert.Equal(400, resp2.StatusCode)
}

func TestRequiredFieldTag(t *testing.T) {
	assert := assert.New(t)

	endpoint := &f.Endpoint[f.NoParams, taggedRequiredBody]{
		Method: "POST",
		Path:   "/required-tag",
		Handler: func(request *f.Request, params f.NoParams, body *taggedRequiredBody) *f.Response {
			return f.Respond.Ok().JSON(body)
		},
	}
	startServer(t, endpoint)

	payload := strings.NewReader(`{"required":"a","optional":"b"}`)
	resp := rawRequest("POST", "http://localhost:8080/required-tag", payload, "application/json")
	assert.Equal(200, resp.StatusCode)

	payload = strings.NewReader(`{"required":"a"}`)
	resp = rawRequest("POST", "http://localhost:8080/required-tag", payload, "application/json")
	assert.Equal(200, resp.StatusCode)

	payload = strings.NewReader(`{"optional":"b"}`)
	resp = rawRequest("POST", "http://localhost:8080/required-tag", payload, "application/json")
	assert.Equal(400, resp.StatusCode)

	payload = strings.NewReader(`{}`)
	resp = rawRequest("POST", "http://localhost:8080/required-tag", payload, "application/json")
	assert.Equal(400, resp.StatusCode)
}

func TestRequiredFormFields(t *testing.T) {
	assert := assert.New(t)

	endpoint := &f.Endpoint[f.NoParams, formBody]{
		Method:              "POST",
		Path:                "/required-form",
		RequireAllBodyField: true,
		Handler: func(request *f.Request, params f.NoParams, body *formBody) *f.Response {
			return f.Respond.Ok().JSON(body)
		},
	}
	startServer(t, endpoint)

	form := url.Values{"name": {"a"}, "req": {"b"}}
	resp := rawRequest("POST", "http://localhost:8080/required-form",
		strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	assert.Equal(200, resp.StatusCode)

	form = url.Values{"req": {"b"}}
	resp = rawRequest("POST", "http://localhost:8080/required-form",
		strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	assert.Equal(400, resp.StatusCode)

	form = url.Values{"name": {"a"}}
	resp = rawRequest("POST", "http://localhost:8080/required-form",
		strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	assert.Equal(400, resp.StatusCode)

	// an empty value still counts as present for form fields
	form = url.Values{"name": {"a"}, "req": {""}}
	resp = rawRequest("POST", "http://localhost:8080/required-form",
		strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	assert.Equal(200, resp.StatusCode)
}

func rawRequest(method string, url string, body io.Reader, contentType string) *http.Response {
	req, err := http.NewRequest(method, url, body)
	noErr(err)
	if body != nil {
		req.Header.Add("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	noErr(err)
	resp.Body.Close()
	return resp
}
