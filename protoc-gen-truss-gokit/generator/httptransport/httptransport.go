// Package httptransport provides functions and template helpers for templating
// the http-transport of a go-kit based microservice.
package httptransport

import (
	"fmt"
	"strings"
)

// PathExtract takes a url, a gRPC-annotation style url template, and a field
// within that url template, and returns the extracted field from the url,
// based on that fields location within the template.
//
// PathExtract does not support the entirety of the URL template syntax defined
// in third_party/googleapis/google/api/httprule.proto. Only a small subset of
// the functionality defined there is implemented here.
func PathExtract(url string, urlTmpl string, field string) (string, error) {
	ErrFieldNotFound := fmt.Errorf("httptransport: field not found in template")
	removeBraces := func(val string) string {
		val = strings.Replace(val, "{", "", -1)
		val = strings.Replace(val, "}", "", -1)
		return val
	}
	buildParamMap := func(urlTmpl string) map[string]int {
		rv := map[string]int{}

		parts := strings.Split(urlTmpl, "/")
		for idx, part := range parts {
			if strings.ContainsAny(part, "{}") {
				param := removeBraces(part)
				rv[param] = idx
			}
		}
		return rv
	}
	pmp := buildParamMap(urlTmpl)

	var loc int
	if val, ok := pmp[field]; !ok {
		return "", ErrFieldNotFound
	} else {
		loc = val
	}
	parts := strings.Split(url, "/")
	return parts[loc], nil
}
