// Package goreq is a simple http client request builder, inspired from https://github.com/dghubble/sling
package goreq

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gookit/goutil/netutil/httpreq"
)

// Middleware interface for client request.
type Middleware interface {
	Handle(*http.Request) *Response
}

// MiddleFunc implements the Middleware interface
type MiddleFunc func(*http.Request) *Response

// Handle request
func (mf MiddleFunc) Handle(r *http.Request) *Response {
	return mf(r)
}

// HReq is an HTTP Request builder and sender.
type HReq struct {
	client httpreq.HttpDoer
	// http method eg: GET,POST
	method  string
	header  http.Header
	baseURL string
	// query structs data
	queryStructs []interface{}
	// body provider
	bodyProvider BodyProvider
	respDecoder  RespDecoder
	// beforeSend callback
	beforeSend func(req *http.Request)
}

// New create
func New() *HReq {
	return &HReq{
		client: http.DefaultClient,
		method: http.MethodGet,
		header: make(http.Header),
		// default use JSON decoder
		respDecoder:  jsonDecoder{},
		queryStructs: make([]interface{}, 0),
	}
}

// New create an instance from current.
func (h *HReq) New() *HReq {
	// copy Headers pairs into new Header map
	headerCopy := make(http.Header)
	for k, v := range h.header {
		headerCopy[k] = v
	}

	return &HReq{
		client:  h.client,
		method:  h.method,
		baseURL: h.baseURL,
		header:  headerCopy,
		// queryStructs:    append([]interface{}{}, s.queryStructs...),
		bodyProvider: h.bodyProvider,
		respDecoder:  h.respDecoder,
	}
}

// Doer custom set http request doer.
// If a nil client is given, the http.DefaultClient will be used.
func (h *HReq) Doer(doer httpreq.HttpDoer) *HReq {
	if doer != nil {
		h.client = doer
	} else {
		h.client = http.DefaultClient
	}

	return h
}

// Client custom set http request doer
func (h *HReq) Client(doer httpreq.HttpDoer) *HReq {
	return h.Doer(doer)
}

// HttpClient custom set http client as request doer
func (h *HReq) HttpClient(hClient *http.Client) *HReq {
	return h.Doer(hClient)
}

// Config custom config http request doer
func (h *HReq) Config(fn func(doer httpreq.HttpDoer)) *HReq {
	fn(h.client)
	return h
}

// ConfigHClient custom config http client.
func (h *HReq) ConfigHClient(fn func(hClient *http.Client)) *HReq {
	if hc, ok := h.client.(*http.Client); ok {
		fn(hc)
	} else {
		panic("the doer is not an *http.Client")
	}

	return h
}

// ------------ Method ------------

// Head sets the method to HEAD and request the pathURL, then return response.
func (h *HReq) Head(pathURL string) (*http.Response, error) {
	return h.Method(http.MethodHead).Send(pathURL)
}

// Get sets the method to GET and sets the given pathURL.
func (h *HReq) Get(pathURL string) (*http.Response, error) {
	return h.Method(http.MethodGet).Send(pathURL)
}

// Method set http method name.
func (h *HReq) Method(method string) *HReq {
	h.method = method
	return h
}

// ----------- URL, query params ------------

// BaseURL set base URL for request
func (h *HReq) BaseURL(baseURL string) *HReq {
	h.baseURL = baseURL
	return h
}

// QueryValues appends url.Values to the query string. The value will be encoded as
// url query parameters on new requests (see Request()).
func (h *HReq) QueryValues(values url.Values) *HReq {
	if values != nil {
		h.queryStructs = append(h.queryStructs, values)
	}

	return h
}

// ----------- Header ------------

// AddHeader adds the key, value pair in Headers, appending values for existing keys
// to the key's values. Header keys are canonicalized.
func (h *HReq) AddHeader(key, value string) *HReq {
	h.header.Add(key, value)
	return h
}

// SetHeader sets the key, value pair in Headers, replacing existing values
// associated with key. Header keys are canonicalized.
func (h *HReq) SetHeader(key, value string) *HReq {
	h.header.Set(key, value)
	return h
}

// AddHeaders adds all the http.Header values, appending values for existing keys
// to the key's values. Header keys are canonicalized.
func (h *HReq) AddHeaders(headers http.Header) *HReq {
	for key, values := range headers {
		for i := range values {
			h.header.Add(key, values[i])
		}
	}
	return h
}

// SetHeaders sets all the http.Header values, replacing values for existing keys
// to the key's values. Header keys are canonicalized.
func (h *HReq) SetHeaders(headers http.Header) *HReq {
	for key, values := range headers {
		for i := range values {
			if i == 0 {
				h.header.Set(key, values[i])
			} else {
				h.header.Add(key, values[i])
			}
		}
	}
	return h
}

// ContentType with custom ContentType header
func (h *HReq) ContentType(value string) *HReq {
	return h.SetHeader("ContentType", value)
}

// BasicAuth sets the Authorization header to use HTTP Basic Authentication
// with the provided username and password. With HTTP Basic Authentication
// the provided username and password are not encrypted.
func (h *HReq) BasicAuth(username, password string) *HReq {
	return h.SetHeader("Authorization", "Basic "+basicAuth(username, password))
}

// basicAuth returns the base64 encoded username:password for basic auth copied
// from net/http.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// ----------- Body ------------

// Body with custom body
func (h *HReq) Body(r io.Reader) *HReq {
	h.bodyProvider = bodyProvider{body: r}
	return h
}

// BytesBody with custom string body
func (h *HReq) BytesBody(bs []byte) *HReq {
	return h.Body(bytes.NewReader(bs))
}

// StringBody with custom string body
func (h *HReq) StringBody(s string) *HReq {
	return h.Body(strings.NewReader(s))
}

// ----------- Do send request ------------

// Send request and return response
func (h *HReq) Send(pathURL string) (*http.Response, error) {
	return h.SendWithCtx(context.Background(), pathURL)
}

// MustSend send request and return response, will panic on error
func (h *HReq) MustSend(pathURL string) *http.Response {
	resp, err := h.SendWithCtx(context.Background(), pathURL)
	if err != nil {
		panic(err)
	}

	return resp
}

// SendWithCtx request with context, then return response
func (h *HReq) SendWithCtx(ctx context.Context, pathURL string) (*http.Response, error) {
	fullURL := pathURL
	if len(h.baseURL) > 0 {
		// pathURL is a not full URL
		if !strings.HasPrefix(pathURL, "http") {
			fullURL = h.baseURL + pathURL
		} else if len(pathURL) == 0 {
			fullURL = h.baseURL
		}
	}

	reqURL, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}

	err = addQueryStructs(reqURL, h.queryStructs)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if h.bodyProvider != nil {
		body, err = h.bodyProvider.Body()
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, h.method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}

	httpreq.AddHeadersToRequest(req, h.header)

	if h.beforeSend != nil {
		h.beforeSend(req)
	}
	return h.client.Do(req)
}

func addQueryStructs(reqURL *url.URL, qss []interface{}) error {
	urlValues, err := url.ParseQuery(reqURL.RawQuery)
	if err != nil {
		return err
	}

	for _, queryStruct := range qss {
		queryValues := httpreq.ToQueryValues(queryStruct)

		for key, values := range queryValues {
			for _, value := range values {
				urlValues.Add(key, value)
			}
		}
	}

	// url.Values format to a sorted "url encoded" string.
	// e.g. "key=val&foo=bar"
	reqURL.RawQuery = urlValues.Encode()
	return nil
}
