package pbinfo

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/TuneLab/go-truss/deftree/svcparse"
	"github.com/pkg/errors"
)

// ConsolidateHTTP accepts a Catalog and the io.Readers for the proto files
// comprising the definition. It modifies the Catalog so that HTTPBindings and
// their associated HTTPParamters are added to each ServiceMethod. After this,
// each `HTTPBinding` will have a populated list of all the http parameters
// that that binding requires, where that parameter should be located, and the
// type of each parameter.
func ConsolidateHTTP(cat *Catalog, protoFiles []io.Reader) error {
	for _, pfile := range protoFiles {
		lex := svcparse.NewSvcLexer(pfile)
		protosvc, err := svcparse.ParseService(lex)
		if err != nil {
			if strings.Contains("'options' or", err.Error()) {
				log.Warnf("Parser found rpc method which lacks HTTP " +
					"annotations; this is allowed, but will result in HTTP " +
					"transport not being generated.")
			} else {
				return errors.Wrap(err, "error while parsing http options for the service definition")
			}
		}
		assembleHTTPParams(cat.Service, protosvc)
	}
	return nil
}

func assembleHTTPParams(svc *Service, httpsvc *svcparse.Service) error {
	getMethNamed := func(name string) *ServiceMethod {
		for _, m := range svc.Methods {
			if m.Name == name {
				return m
			}
		}
		return nil
	}

	createParams := func(meth *ServiceMethod, parsedbind *svcparse.HTTPBinding) {
		msg := meth.RequestType.Message
		bind := HTTPBinding{}
		bind.Verb, bind.Path = getVerb(parsedbind)

		params := make([]*HTTPParameter, 0)
		for _, field := range msg.Fields {
			new_param := &HTTPParameter{}
			new_param.Field = field
			new_param.Location = paramLocation(field, parsedbind)
			params = append(params, new_param)
		}
		bind.Params = params
		meth.Bindings = append(meth.Bindings, &bind)
	}

	for _, hm := range httpsvc.Methods {
		m := getMethNamed(hm.Name)
		if m == nil {
			return errors.New(fmt.Sprintf("Could not find service method named %q", hm.Name))
		}
		for _, hbind := range hm.HTTPBindings {
			createParams(m, hbind)
		}
	}
	return nil
}

// getVerb returns the verb of a svcparse.HTTPBinding. If the binding does not
// contain a field with a verb, returns two empty strings. Currently doesn't
// support "custom" verbs.
func getVerb(binding *svcparse.HTTPBinding) (verb string, path string) {
	for _, field := range binding.Fields {
		switch field.Kind {
		case "get", "put", "post", "delete", "patch":
			return field.Kind, field.Value
		}
	}
	return "", ""
}

// paramLocation returns the location that a field would be found according to
// the rules of a given HTTPBinding.
func paramLocation(field *Field, binding *svcparse.HTTPBinding) string {
	path_params := getPathParams(binding)
	for _, path_param := range path_params {
		if strings.Split(path_param, ".")[0] == field.Name {
			return "path"
		}
	}
	for _, optField := range binding.Fields {
		if optField.Kind == "body" {
			if optField.Value == "*" {
				return "body"
			} else if optField.Value == field.Name {
				return "body"
			}
		}
	}

	return "query"
}

// Returns a slice of strings containing all parameters in the path
func getPathParams(binding *svcparse.HTTPBinding) []string {
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
