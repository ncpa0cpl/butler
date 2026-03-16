package butler

import (
	"strings"
)

type genericHeaderCollection interface {
	Get(name string) string
	Set(name string, value string)
	Add(name string, value string)
	Del(key string)
}

type H struct {
	Name   string
	Values []string
}

type Headers struct {
	httpHeaders []H
}

func NewHeaders(initHeaders ...H) *Headers {
	h := &Headers{
		httpHeaders: initHeaders,
	}
	return h
}

func (h *Headers) Has(name string) bool {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		h := &h.httpHeaders[idx]
		if h.Name == name && len(h.Values) > 0 {
			return true
		}
	}

	return false
}

func (h *Headers) Get(name string) string {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		h := &h.httpHeaders[idx]
		if h.Name == name {
			return h.Values[len(h.Values)-1]
		}
	}

	return ""
}

func (h *Headers) Set(name string, value string) *Headers {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		header := &h.httpHeaders[idx]
		if header.Name == name {
			header.Values = []string{value}
			return h
		}
	}

	h.httpHeaders = append(h.httpHeaders, H{name, []string{value}})

	return h
}

func (h *Headers) Add(name string, value string) *Headers {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		header := &h.httpHeaders[idx]
		if header.Name == name {
			header.Values = append(header.Values, value)
			return h
		}
	}

	h.httpHeaders = append(h.httpHeaders, H{name, []string{value}})

	return h
}

func (h *Headers) Del(name string) *Headers {
	name = strings.ToLower(name)

	for idx := range h.httpHeaders {
		if h.httpHeaders[idx].Name == name {
			h.httpHeaders[idx] = h.httpHeaders[len(h.httpHeaders)-1]
			h.httpHeaders = h.httpHeaders[:len(h.httpHeaders)-1]
			return h
		}
	}

	return h
}

/*
Copies all the headers set on this butler.Headers instance into the target.
The target must be either another butler.Headers or a http.Header.
*/
func (h *Headers) CopyInto(target any) {
	if headers, ok := target.(*Headers); ok {
		for idx := range h.httpHeaders {
			header := &h.httpHeaders[idx]

			if len(header.Values) == 0 {
				continue
			}

			headers.Del(header.Name)

			if len(header.Values) == 1 {
				headers.Set(header.Name, header.Values[0])
				continue
			}

			for _, value := range header.Values {
				headers.Add(header.Name, value)
			}
		}
	} else if headers, ok := target.(genericHeaderCollection); ok {
		for idx := range h.httpHeaders {
			header := &h.httpHeaders[idx]

			if len(header.Values) == 0 {
				continue
			}

			headers.Del(header.Name)

			if len(header.Values) == 1 {
				headers.Set(header.Name, header.Values[0])
				continue
			}

			for _, value := range header.Values {
				headers.Add(header.Name, value)
			}
		}
	} else {
		panic("Headers.CopyInto target is not a valid header collection")
	}
}
