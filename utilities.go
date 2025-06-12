package butler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"hash/fnv"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/gabriel-vasile/mimetype"
)

var ENCODABLE_MIME_TYPES []string = []string{
	"application/json",
	"application/xml",
	"application/yaml",
	"text/calendar",
	"text/css",
	"text/csv",
	"text/html",
	"text/javascript",
	"text/markdown",
	"text/mathml",
	"text/plain",
	"text/prs.lines.tag",
	"text/richtext",
	"text/sgml",
	"text/tab-separated-values",
	"text/troff",
	"text/uri-list",
}

const BROTLI_MIN_SIZE = 256
const DEFLATE_MIN_SIZE = 512
const GZIP_MIN_SIZE = 1024

func canEncode(contentType string) bool {
	for _, mimeType := range ENCODABLE_MIME_TYPES {
		if strings.Contains(contentType, mimeType) {
			return true
		}
	}
	return false
}

func resolveAutoEncoding(request *Request, response *Response) string {
	respContentType := response.Headers.Get("Content-Type")

	if !canEncode(respContentType) {
		return "none"
	}

	acceptedEncodings := request.Headers.Get("Accept-Encoding")
	bodyLen := len(response.Body)

	if strings.Contains(acceptedEncodings, "br") {
		if bodyLen >= BROTLI_MIN_SIZE {
			return "brotli"
		}
	}

	if strings.Contains(acceptedEncodings, "deflate") {
		if bodyLen >= DEFLATE_MIN_SIZE {
			return "deflate"
		}
	}

	if strings.Contains(acceptedEncodings, "gzip") {
		if bodyLen >= GZIP_MIN_SIZE {
			return "gzip"
		}
	}

	return "none"
}

func GZip(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}
	return &buf, writer.Close()
}

func Deflate(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	writer, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}
	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}
	return &buf, writer.Close()
}

func Brotli(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	writer := brotli.NewWriter(&buf)
	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}
	return &buf, writer.Close()
}

func EncodeRequestGzip(request *Request, resp *Response) error {
	if len(resp.Body) >= GZIP_MIN_SIZE && resp.Headers.Get("Content-Encoding") == "" {
		acceptedEncodings := request.Headers.Get("Accept-Encoding")
		if strings.Contains(acceptedEncodings, "gzip") {
			data, err := GZip(resp.Body)
			if err == nil {
				resp.Body = data.Bytes()
				resp.Headers.Set("Content-Encoding", "gzip")
			} else {
				return fmt.Errorf("encountered an error when encoding the response (GZip)")
			}
		}
	}
	return nil
}

func EncodeRequestDeflate(request *Request, resp *Response) error {
	if len(resp.Body) >= DEFLATE_MIN_SIZE && resp.Headers.Get("Content-Encoding") == "" {
		acceptedEncodings := request.Headers.Get("Accept-Encoding")
		if strings.Contains(acceptedEncodings, "deflate") {
			data, err := Deflate(resp.Body)
			if err == nil {
				resp.Body = data.Bytes()
				resp.Headers.Set("Content-Encoding", "deflate")
			} else {
				return fmt.Errorf("encountered an error when encoding the response (Deflate)")
			}
		}
	}
	return nil
}

func EncodeRequestBrotli(request *Request, resp *Response) error {
	if len(resp.Body) >= BROTLI_MIN_SIZE && resp.Headers.Get("Content-Encoding") == "" {
		acceptedEncodings := request.Headers.Get("Accept-Encoding")
		if strings.Contains(acceptedEncodings, "br") {
			data, err := Brotli(resp.Body)
			if err == nil {
				resp.Body = data.Bytes()
				resp.Headers.Set("Content-Encoding", "br")
			} else {
				return fmt.Errorf("encountered an error when encoding the response (Brotli)")
			}
		}
	}
	return nil
}

func AddEtag(response *Response) {
	if len(response.Body) == 0 {
		return
	}

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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

type Range struct {
	HasStart bool
	HasEnd   bool
	Start    int
	End      int
}

func parseRangeHeader(headers genericHeaderCollection) (*Range, error) {
	header := headers.Get("Range")

	if len(header) == 0 || !strings.HasPrefix(header, "bytes=") {
		return nil, nil
	}

	header = strings.TrimPrefix(header, "bytes=")

	rangeParts := strings.Split(header, "-")

	r := &Range{}

	if len(rangeParts) > 0 && rangeParts[0] != "" {
		startValue, err := strconv.ParseUint(rangeParts[0], 10, 64)

		if err == nil {
			r.Start = int(startValue)
			r.HasStart = true
		} else {
			return nil, err
		}
	}

	if len(rangeParts) > 1 && rangeParts[1] != "" {
		endValue, err := strconv.ParseUint(rangeParts[1], 10, 64)

		if err == nil {
			r.End = int(endValue)
			r.HasEnd = true
		} else {
			return nil, err
		}
	}

	return r, nil
}

func pathJoin(a, b string) string {
	if b == "" {
		return a
	}

	return strings.TrimRight(a, "/") + "/" + strings.TrimLeft(b, "/")
}

func firstOr[T any](slice []T, defaultV T) T {
	if len(slice) > 0 {
		return slice[0]
	}
	return defaultV
}

type mimet struct{}

var Mime mimet

func (mimet) DetectFile(filepath string, file *os.File) string {
	ext := strings.ToLower(path.Ext(file.Name()))
	switch ext {
	case ".js":
		return "text/javascript"
	case ".json":
		return "application/json"
	case ".css":
		return "text/css"
	case ".xml":
		return "application/xml"
	case ".html":
		return "text/html"
	case ".md":
		return "text/markdown"
	case ".yaml", ".yml":
		return "application/yaml"
	case ".csv":
		return "text/csv"
	}

	t, err := mimetype.DetectReader(file)
	if err != nil {
		return "application/octet-stream"
	}

	return t.String()
}

type units struct {
	KB int64
	MB int64
	GB int64
}

var Units units = units{
	KB: 1024,
	MB: 1024 * 1024,
	GB: 1024 * 1024 * 1024,
}
