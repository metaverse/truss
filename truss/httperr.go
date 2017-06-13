package truss

import "net/http"

// HTTPError encodes information in an error type for the
// github.com/go-kit/kit/tranport/http.DefaultErrorEncoder to create a
// meaningful http response
func HTTPError(err error, statusCode int, headers http.Header) error {
	return httpError{
		err,
		statusCode,
		headers,
	}
}

type httpError struct {
	error
	statusCode int
	headers    map[string][]string
}

func (h httpError) StatusCode() int {
	return h.statusCode
}

func (h httpError) Headers() http.Header {
	return h.headers
}
