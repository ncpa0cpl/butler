package butler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

func waitUntil(predicate func() bool) {
	start := time.Now()

	for !predicate() {
		if time.Since(start).Seconds() > 10 {
			panic("predicate not satisfied within the timeout")
		}

		time.Sleep(200 * time.Millisecond)
	}
}

type header struct {
	name  string
	value string
}

func request(method string, url string, body any, headers ...header) ([]byte, *http.Response) {
	client := http.Client{}

	var payload io.Reader
	if body != nil {
		if v, ok := body.([]byte); ok {
			payload = bytes.NewBuffer(v)
		} else {
			postBody, err := json.Marshal(body)
			noErr(err)
			payload = bytes.NewBuffer(postBody)
		}
	}
	req, err := http.NewRequest(method, url, payload)
	noErr(err)
	req.Close = true
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}
	for _, h := range headers {
		req.Header.Add(h.name, h.value)
	}

	resp, err := client.Do(req)
	noErr(err)

	respBody, err := io.ReadAll(resp.Body)
	noErr(err)

	return respBody, resp
}

func noErr(err error) {
	if err != nil {
		panic(err)
	}
}
