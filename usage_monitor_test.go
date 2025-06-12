package butler_test

import (
	"net/http"
	"testing"
	"time"

	f "github.com/ncpa0cpl/butler"
	"github.com/stretchr/testify/assert"
)

type TestMonitor struct {
	records []f.UsageRecord
}

func (tm *TestMonitor) Record(entry *f.UsageRecord) {
	tm.records = append(tm.records, *entry)
}

func TestUsageMonitor(t *testing.T) {
	assert := assert.New(t)

	server := f.CreateServer()
	server.Port = 8080

	monitor := TestMonitor{}
	server.Monitor(&monitor)

	books := &f.BasicEndpoint[f.NoParams]{
		Method:   "GET",
		Path:     "/books",
		Encoding: "gzip",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
		},
		Auth: func(request *f.Request) *f.Ath {
			return f.Auth.Ok()
		},
		Handler: func(request *f.Request, params f.NoParams) *f.Response {
			return f.Respond.Ok().Html(HTML_SAMPLE)
		},
	}

	group := &f.Group{
		Path: "/api",
	}

	group.Use(f.Middleware{
		Name: "MdFoo",
		OnRequest: func(request *f.Request, respond func(response *f.Response)) error {
			return nil
		},
		OnResponse: func(request *f.Request, response *f.Response, next func(response *f.Response)) error {
			return nil
		},
	})

	group.Add(books)
	server.Add(group)

	go server.Listen()
	defer server.Close()

	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://localhost:8080/api/books", nil)
	noErr(err)
	req.Header.Set("accept-encoding", "gzip")
	resp, err := client.Do(req)
	noErr(err)
	assert.Equal(200, resp.StatusCode)

	waitUntil(func() bool {
		return len(monitor.records) == 1
	})

	assert.Equal("/api/books", monitor.records[0].UrlPath)
	assert.NotNil(monitor.records[0].Start)
	assert.NotNil(monitor.records[0].End)
	assert.Equal(6, len(monitor.records[0].Steps))

	authStep := monitor.records[0].Steps[0]
	reqMdStep := monitor.records[0].Steps[1]
	handlerStem := monitor.records[0].Steps[2]
	resMdStep := monitor.records[0].Steps[3]
	etagStep := monitor.records[0].Steps[4]
	encodeStep := monitor.records[0].Steps[5]

	assert.Equal("auth", authStep.Step)
	assert.Equal("", authStep.Name)
	assert.NotNil(authStep.Start)
	assert.NotNil(authStep.End)

	assert.Equal("middleware:request", reqMdStep.Step)
	assert.Equal("MdFoo", reqMdStep.Name)
	assert.NotNil(reqMdStep.Start)
	assert.NotNil(reqMdStep.End)

	assert.Equal("handler", handlerStem.Step)
	assert.Equal("", handlerStem.Name)
	assert.NotNil(handlerStem.Start)
	assert.NotNil(handlerStem.End)

	assert.Equal("middleware:response", resMdStep.Step)
	assert.Equal("MdFoo", resMdStep.Name)
	assert.NotNil(resMdStep.Start)
	assert.NotNil(resMdStep.End)

	assert.Equal("internal:etag", etagStep.Step)
	assert.Equal("", etagStep.Name)
	assert.NotNil(etagStep.Start)
	assert.NotNil(etagStep.End)

	assert.Equal("internal:encoding", encodeStep.Step)
	assert.Equal("", encodeStep.Name)
	assert.NotNil(encodeStep.Start)
	assert.NotNil(encodeStep.End)
}
