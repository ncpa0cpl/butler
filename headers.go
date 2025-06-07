package httpbutler

import "strings"

type genericHeaderCollection interface {
	Get(name string) string
	Set(name string, value string)
	Add(name string, value string)
	Del(key string)
}

type header struct {
	name   string
	values []string
}

type Headers struct {
	httpHeaders []header
}

func (h *Headers) Has(name string) bool {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		h := &h.httpHeaders[idx]
		if h.name == name && len(h.values) > 0 {
			return true
		}
	}

	return false
}

func (h *Headers) Get(name string) string {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		h := &h.httpHeaders[idx]
		if h.name == name {
			return h.values[len(h.values)-1]
		}
	}

	return ""
}

func (h *Headers) Set(name string, value string) {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		h := &h.httpHeaders[idx]
		if h.name == name {
			h.values = []string{value}
			return
		}
	}

	h.httpHeaders = append(h.httpHeaders, header{name, []string{value}})
}

func (h *Headers) Add(name string, value string) {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		h := &h.httpHeaders[idx]
		if h.name == name {
			h.values = append(h.values, value)
			return
		}
	}

	h.httpHeaders = append(h.httpHeaders, header{name, []string{value}})
}

func (h *Headers) Del(name string) {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		if h.httpHeaders[idx].name == name {
			h.httpHeaders[idx] = h.httpHeaders[len(h.httpHeaders)-1]
			h.httpHeaders = h.httpHeaders[:len(h.httpHeaders)-1]
			return
		}
	}
}

func (h *Headers) CopyInto(target genericHeaderCollection) {
	for idx := range h.httpHeaders {
		header := &h.httpHeaders[idx]

		if len(header.values) == 0 {
			continue
		}

		target.Del(header.name)

		if len(header.values) == 1 {
			target.Set(header.name, header.values[0])
			continue
		}

		for _, value := range header.values {
			target.Add(header.name, value)
		}
	}
}
