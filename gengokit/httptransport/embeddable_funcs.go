package httptransport

import (
	"encoding/json"
	"fmt"
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

	expectedLen := len(strings.Split(strings.TrimRight(urlTmpl, "/"), "/"))
	recievedLen := len(strings.Split(strings.TrimRight(url, "/"), "/"))
	if expectedLen != recievedLen {
		return nil, fmt.Errorf("expecting a path containing %d parts, provided path contains %d parts", expectedLen, recievedLen)
	}

	parts := strings.Split(url, "/")
	for k, v := range pmp {
		rv[k] = parts[v]
	}

	return rv, nil
}

// BuildParamMap takes a string representing a url template and returns a map
// indicating the location of each parameter within that url, where the
// location is the index as if in a slash-separated sequence of path
// components. For example, given the url template:
//
//     "/v1/{a}/{b}"
//
// The returned param map would look like:
//
//     map[string]int {
//         "a": 2,
//         "b": 3,
//     }
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

// RemoveBraces replace all curly braces in the provided string, opening and
// closing, with empty strings.
func RemoveBraces(val string) string {
	val = strings.Replace(val, "{", "", -1)
	val = strings.Replace(val, "}", "", -1)
	return val
}

// encodePathParams encodes `mux.Vars()` with dot notations into JSON objects
// to be unmarshaled into non-basetype fields.
// e.g. {"book.name": "books/1"} -> {"book": {"name": "books/1"}}
func encodePathParams(vars map[string]string) map[string]string {
	var recur func(path, value string, data map[string]interface{})
	recur = func(path, value string, data map[string]interface{}) {
		parts := strings.SplitN(path, ".", 2)
		key := parts[0]
		if len(parts) == 1 {
			data[key] = value
		} else {
			if _, ok := data[key]; !ok {
				data[key] = make(map[string]interface{})
			}
			recur(parts[1], value, data[key].(map[string]interface{}))
		}
	}

	data := make(map[string]interface{})
	for key, val := range vars {
		recur(key, val, data)
	}

	ret := make(map[string]string)
	for key, val := range data {
		switch val := val.(type) {
		case string:
			ret[key] = val
		case map[string]interface{}:
			m, _ := json.Marshal(val)
			ret[key] = string(m)
		}
	}
	return ret
}
