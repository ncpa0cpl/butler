package httpbutler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	f "github.com/ncpa0cpl/http-butler"
	"github.com/stretchr/testify/assert"
)

func TestGetEndpointWithQueryParams(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	books := &f.BasicEndpoint[QParams]{
		Method: "GET",
		Path:   "/books",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params QParams) *f.Response {
			search := params.Search.Get()
			limit := params.Limit.Get()
			del := params.IncludeDel.Get()

			return f.Respond.Ok().JSON([]Book{
				{
					Title: search,
				},
				{
					Title: fmt.Sprintf("%v", limit),
				},
				{
					Title: fmt.Sprintf("%v", del),
				},
			})
		},
	}

	server.Add(books)

	go server.Listen()
	defer server.Close()

	resp, err := http.Get("http://localhost:8080/books")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	cp := resp.Header.Get("Content-Type")
	assert.Equal("application/json; charset=utf-8", cp)

	cacheControl := resp.Header.Get("Cache-Control")
	assert.Equal("public, max-age=3600", cacheControl)

	etag := resp.Header.Get("ETag")
	assert.NotZero(etag)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("[{\"Title\":\"\"},{\"Title\":\"0\"},{\"Title\":\"false\"}]", string(body))

	resp, err = http.Get("http://localhost:8080/books?search=fooqux&limit=2&includedel=true")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	cp = resp.Header.Get("Content-Type")
	assert.Equal("application/json; charset=utf-8", cp)

	body, err = io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("[{\"Title\":\"fooqux\"},{\"Title\":\"2\"},{\"Title\":\"true\"}]", string(body))

	resp, err = http.Get("http://localhost:8080/books?search=fooqux&limit=a&includedel=true")
	assert.NoError(err)
	assert.Equal(400, resp.StatusCode)
}

func TestGetEndpointWithUrlParams(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	type UrlParams struct {
		ID   *f.StringUrlParam
		Page *f.NumberUrlParam
	}

	books := &f.BasicEndpoint[UrlParams]{
		Method: "GET",
		Path:   "/books/:id/:page",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params UrlParams) *f.Response {
			id := params.ID.Get()
			page := params.Page.Get()

			return f.Respond.Ok().JSON([]Book{
				{
					Title: id,
				},
				{
					Title: fmt.Sprintf("%v", page),
				},
			})
		},
	}

	server.Add(books)

	go server.Listen()
	defer server.Close()

	resp, err := http.Get("http://localhost:8080/books/B1Y332O/5")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	cp := resp.Header.Get("Content-Type")
	assert.Equal("application/json; charset=utf-8", cp)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("[{\"Title\":\"B1Y332O\"},{\"Title\":\"5\"}]", string(body))

	resp, err = http.Get("http://localhost:8080/books/B1Y332O/A")
	assert.NoError(err)
	assert.Equal(400, resp.StatusCode)
}

func TestGroupedEndpoints(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	apiGroup := &f.Group{
		Path: "/api",
	}

	type LoopbackPayload struct {
		Value string
	}

	loopback := &f.Endpoint[f.NoParams, LoopbackPayload]{
		Method: "POST",
		Path:   "/loopback",
		Handler: func(request *f.Request, params f.NoParams, body *LoopbackPayload) *f.Response {
			return f.Respond.Ok().JSON(body)
		},
	}

	apiGroup.Add(loopback)
	server.Add(apiGroup)

	go server.Listen()
	defer server.Close()

	postBody, _ := json.Marshal(&LoopbackPayload{Value: "return this back"})
	resp, err := http.Post("http://localhost:8080/api/loopback", "application/json", bytes.NewBuffer(postBody))
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("{\"Value\":\"return this back\"}", string(body))
}

func TestNestedGroups(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	g1 := &f.Group{
		Path: "/group1",
	}

	g2 := &f.Group{
		Path: "/group2",
	}

	g3 := &f.Group{
		Path: "/group3",
	}

	type LoopbackPayload struct {
		Value string
	}

	loopback := &f.Endpoint[f.NoParams, LoopbackPayload]{
		Method: "POST",
		Path:   "/loopback",
		Handler: func(request *f.Request, params f.NoParams, body *LoopbackPayload) *f.Response {
			return f.Respond.Ok().JSON(body)
		},
	}

	g1.Add(loopback)
	g2.Add(g1)
	g3.Add(g2)
	server.Add(g3)

	go server.Listen()
	defer server.Close()

	postBody, _ := json.Marshal(&LoopbackPayload{Value: "return this back"})
	resp, err := http.Post("http://localhost:8080/group3/group2/group1/loopback", "application/json", bytes.NewBuffer(postBody))
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)

	assert.Equal("{\"Value\":\"return this back\"}", string(body))
}

func TestEtagCaching(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	books := &f.BasicEndpoint[f.NoParams]{
		Method: "GET",
		Path:   "/books",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Handler: func(request *f.Request, params f.NoParams) *f.Response {
			return f.Respond.Ok().JSON([]Book{
				{
					Title: "Murder in Orient Express",
				},
				{
					Title: "It",
				},
				{
					Title: "Harry Potter",
				},
			})
		},
	}

	server.Add(books)

	go server.Listen()
	defer server.Close()

	resp, err := http.Get("http://localhost:8080/books")
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

	etag := resp.Header.Get("ETag")
	assert.NotZero(etag)

	client := &http.Client{}
	request, err := http.NewRequest("GET", "http://localhost:8080/books", nil)
	assert.NoError(err)
	request.Header.Set("If-None-Match", etag)
	resp, err = client.Do(request)
	assert.NoError(err)

	assert.Equal(304, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(err)
	assert.Equal(0, len(body))
}
