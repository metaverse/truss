package httptransport

import (
	"net/url"
	"strings"
)

// Contains all the functions which must be used within templates. Stored all
// together in this file, the string form of their source code can be extracted
// with the function `AllFuncSourceCode`. All these functions are kept here so
// they can be properly tested, while still allowing them to be conveniently
// templated into the code.

// PathParams takes a url and a gRPC-annotation style url template, and
// returns a map of the named parameters in the template and their values in
// the given url.
//
// PathParams does not support the entirety of the URL template syntax defined
// in third_party/googleapis/google/api/httprule.proto. Only a small subset of
// the functionality defined there is implemented here.
func PathParams(url string, urlTmpl string) (map[string]string, error) {
	rv := map[string]string{}
	pmp := BuildParamMap(urlTmpl)

	parts := strings.Split(url, "/")
	for k, v := range pmp {
		rv[k] = parts[v]
	}

	return rv, nil
}

// Given a url template, create a map of the
func BuildParamMap(urlTmpl string) map[string]int {
	rv := map[string]int{}

	parts := strings.Split(urlTmpl, "/")
	for idx, part := range parts {
		if strings.ContainsAny(part, "{}") {
			param := RemoveBraces(part)
			rv[param] = idx
		}
	}
	return rv
}

// Given a string, replace all curly braces, opening and closing, with empty
// strings.
func RemoveBraces(val string) string {
	val = strings.Replace(val, "{", "", -1)
	val = strings.Replace(val, "}", "", -1)
	return val
}

func QueryParams(vals url.Values) (map[string]string, error) {
	// TODO make this not flatten the query params
	// WARNING this is a super huge hack and will ignore repeated values in the
	// query parameter. This should absolutely be correctly implemented later
	// by someone else or maybe future me...
	rv := map[string]string{}
	for k, v := range vals {
		rv[k] = v[0]
	}
	return rv, nil
}
