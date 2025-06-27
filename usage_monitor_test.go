package butler_test

import (
	"sync"
	"testing"
	"time"

	f "github.com/ncpa0cpl/butler"
	"github.com/stretchr/testify/assert"
)

type TestMonitor struct {
	mutex   sync.Mutex
	records []f.UsageRecord
}

func (tm *TestMonitor) Record(entry *f.UsageRecord) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.records = append(tm.records, *entry)
}

func (tm *TestMonitor) Len() int {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	return len(tm.records)
}

func (tm *TestMonitor) Get(idx int) f.UsageRecord {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	return tm.records[idx]
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

	bookById := &f.BasicEndpoint[f.NoParams]{
		Method:   "GET",
		Path:     "/books/:id",
		Encoding: "gzip",
		CachePolicy: &f.HttpCachePolicy{
			MaxAge: time.Hour,
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
	group.Add(bookById)
	server.Add(group)

	go server.Listen()
	defer server.Close()

	_, resp := request("GET", "http://localhost:8080/api/books", nil, header{"accept-encoding", "gzip"})
	assert.Equal(200, resp.StatusCode)

	waitUntil(func() bool {
		return monitor.Len() == 1
	})

	record1 := monitor.Get(0)

	assert.Equal("GET", record1.Method)
	assert.Equal("/api/books", record1.UrlPath)
	assert.NotNil(record1.Start)
	assert.NotNil(record1.End)
	assert.Equal(6, len(record1.Steps))

	authStep := record1.Steps[0]
	reqMdStep := record1.Steps[1]
	handlerStem := record1.Steps[2]
	resMdStep := record1.Steps[3]
	etagStep := record1.Steps[4]
	encodeStep := record1.Steps[5]

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

	_, resp = request("GET", "http://localhost:8080/api/books/12", nil, header{"accept-encoding", "gzip"})
	assert.Equal(200, resp.StatusCode)

	waitUntil(func() bool {
		return monitor.Len() == 2
	})

	record2 := monitor.Get(1)

	assert.Equal("/api/books/12", record2.UrlPath)
	assert.Equal("/api/books/:id", record2.PathPattern)
}
