package goreq

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"

	"github.com/gookit/goutil/netutil/httpctype"
)

// BodyProvider provides Body content for http.Request attachment.
type BodyProvider interface {
	// ContentType returns the Content-Type of the body.
	ContentType() string
	// Body returns the io.Reader body.
	Body() (io.Reader, error)
}

// bodyProvider provides the wrapped body value as a Body for reqests.
type bodyProvider struct {
	body io.Reader
}

func (p bodyProvider) ContentType() string {
	return ""
}

func (p bodyProvider) Body() (io.Reader, error) {
	return p.body, nil
}

// jsonBodyProvider encodes a JSON tagged struct value as a Body for requests.
type jsonBodyProvider struct {
	payload interface{}
}

// ContentType value
func (p jsonBodyProvider) ContentType() string {
	return httpctype.JSON
}

// Body ger body reader
func (p jsonBodyProvider) Body() (io.Reader, error) {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(p.payload)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// formBodyProvider encodes a url tagged struct value as Body for requests.
// See https://godoc.org/github.com/google/go-querystring/query for details.
type formBodyProvider struct {
	payload interface{}
}

func (p formBodyProvider) ContentType() string {
	return httpctype.Form
}

func (p formBodyProvider) Body() (io.Reader, error) {
	values, ok := p.payload.(url.Values)
	if ok {
		return strings.NewReader(values.Encode()), nil
	}

	if str, ok := p.payload.(string); ok {
		return strings.NewReader(str), nil
	}

	return nil, errors.New("invalid playload for form data")
}
