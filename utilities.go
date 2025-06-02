package httpbutler

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"hash/fnv"
)

func GZip(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}
	return &buf, writer.Close()
}

func AddEtag(response *Response) {
	if response.etag != "" {
		response.Headers.Set("ETag", response.etag)
		return
	}

	if response.Headers.Get("ETag") == "" {
		h := fnv.New64a()
		h.Write(response.Body)
		hashValue := h.Sum64()
		etag := fmt.Sprintf("%x", hashValue)
		response.Headers.Set("ETag", etag)
	}
}
