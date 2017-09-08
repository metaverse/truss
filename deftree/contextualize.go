package deftree

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Assemble takes a deftree that's already had http options parsed by svcparse
// and inserted, then assembles the `HttpParameters` corresponding to each
// ServiceMethod's http annotations. After this, each `HttpBinding` will have a
// populated list of all the http parameters that that binding requires, where
// that parameter should be located, and the type of each parameter.
func Assemble(dt Deftree) error {
	md := dt.(*MicroserviceDefinition)
	for _, file := range md.Files {
		for _, svc := range file.Services {
			for _, meth := range svc.Methods {
				for _, pbind := range meth.HttpBindings {
					err := contextualizeBinding(meth, pbind)
					if err != nil {
						return errors.Wrap(err, "contextualizing http bindings failed")
					}
				}
			}
		}
	}

	return nil
}

func contextualizeBinding(meth *ServiceMethod, binding *MethodHttpBinding) error {
	msg := meth.RequestType

	// Find the verb and the path
	binding.Verb, binding.Path = getVerb(binding)

	params := make([]*HttpParameter, 0)
	// Create the new HttpParameters
	for _, field := range msg.Fields {
		new_param := &HttpParameter{}
		new_param.Name = field.Name
		new_param.Type = field.Type.GetName()
		new_param.Location = paramLocation(field, binding)
		params = append(params, new_param)
	}
	binding.Params = params
	return nil
}

// Get's the verb of binding. Currently doesn't support "custom" verbs.
func getVerb(binding *MethodHttpBinding) (verb string, path string) {
	if binding.CustomHTTPPattern != nil {
		for _, field := range binding.CustomHTTPPattern {
			if field.Kind == "kind" {
				verb = field.Value
			} else if field.Kind == "path" {
				path = field.Value
			}
		}
		return verb, path
	}
	for _, field := range binding.Fields {
		switch field.Kind {
		case "get", "put", "post", "delete", "patch":
			return field.Kind, field.Value
		}
	}
	return "", ""
}

func paramLocation(field *MessageField, binding *MethodHttpBinding) string {
	path_params := getPathParams(binding)
	for _, path_param := range path_params {
		if strings.Split(path_param, ".")[0] == field.GetName() {
			return "path"
		}
	}
	for _, optField := range binding.Fields {
		if optField.Kind == "body" {
			if optField.Value == "*" {
				return "body"
			} else if optField.Value == field.GetName() {
				return "body"
			}
		}
	}

	return "query"
}

// Returns a slice of strings containing all parameters in the path
func getPathParams(binding *MethodHttpBinding) []string {
	_, path := getVerb(binding)
	find_params := regexp.MustCompile("{(.*?)}")
	remove_braces := regexp.MustCompile("{|}")
	params := find_params.FindAllString(path, -1)
	rv := []string{}
	for _, p := range params {
		rv = append(rv, remove_braces.ReplaceAllString(p, ""))
	}
	return rv
}
