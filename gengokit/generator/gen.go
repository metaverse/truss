// Package generator generates a gokit service based on a service definition.
package generator

import (
	"go/format"
	"io"
	"io/ioutil"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit"
	"github.com/TuneLab/go-truss/gengokit/handler"
	templFiles "github.com/TuneLab/go-truss/gengokit/template"

	"github.com/TuneLab/go-truss/svcdef"
	"github.com/TuneLab/go-truss/truss"
)

// GenerateGokit returns a gokit service generated from a service definition (svcdef),
// the package to the root of the generated service goPackage, the package
// to the .pb.go service struct files (goPBPackage) and any prevously generated files.
func GenerateGokit(sd *svcdef.Svcdef, conf gengokit.Config) ([]truss.NamedReadWriter, error) {
	executor, err := gengokit.NewExecutor(sd, conf)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create template executor")
	}

	prevFiles := make(map[string]io.Reader, len(conf.PreviousFiles))
	for _, f := range conf.PreviousFiles {
		prevFiles[f.Name()] = f
	}

	var codeGenFiles []truss.NamedReadWriter

	for _, templFP := range templFiles.AssetNames() {
		actualFP := templatePathToActual(templFP, sd.PkgName)
		prev := prevFiles[actualFP]
		file, err := generateResponseFile(executor, prev, templFP)
		if err != nil {
			return nil, errors.Wrap(err, "cannot render template")
		}
		if file == nil {
			continue
		}

		codeGenFiles = append(codeGenFiles, file)
	}

	return codeGenFiles, nil
}

// generateResponseFile contains logic to choose how to render a template file
// based on path and if that file was generated previously. It accepts a
// template path to render, a executor to apply to the template,
// and . It returns a truss.NamedReadWriter representing the generated file.
func generateResponseFile(executor *gengokit.Executor, file io.Reader, templFP string) (truss.NamedReadWriter, error) {
	var genCode io.Reader
	var err error

	// Get the actual path to the file rather than the template file path
	actualFP := templatePathToActual(templFP, executor.PackageName)

	switch templFP {
	case handler.ServerFile:
		h, err := handler.New(executor.Service, file, executor.PackageName)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot parse previous handler: %q", actualFP)
		}

		if genCode, err = h.Render(templFP, executor); err != nil {
			return nil, errors.Wrapf(err, "cannot render template: %s", templFP)
		}
	default:
		if genCode, err = applyTemplateFromPath(templFP, executor); err != nil {
			return nil, errors.Wrapf(err, "cannot render template: %s", templFP)
		}
	}

	codeBytes, err := ioutil.ReadAll(genCode)
	if err != nil {
		return nil, err
	}

	// ignore error as we want to write the code either way to inspect after
	// writing to disk
	formattedCode := formatCode(codeBytes)

	var resp truss.SimpleFile

	// Set the path to the file and write the code to the file
	resp.Path = actualFP
	if _, err = resp.Write(formattedCode); err != nil {
		return nil, err
	}

	return &resp, nil
}

// templatePathToActual accepts a templateFilePath and the packageName of the
// service and returns what the relative file path of what should be written to
// disk
func templatePathToActual(templFilePath, packageName string) string {
	// Switch "NAME" in path with packageName.
	// i.e. for packageName = addsvc; /NAME-service/NAME-server -> /addsvc-service/addsvc-server
	actual := strings.Replace(templFilePath, "NAME", packageName, -1)

	actual = strings.TrimSuffix(actual, "template")

	return actual
}

// applyTemplateFromPath calls applyTemplate with the template at templFilePath
func applyTemplateFromPath(templFP string, executor *gengokit.Executor) (io.Reader, error) {
	templBytes, err := templFiles.Asset(templFP)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find template file: %v", templFP)
	}

	return executor.ApplyTemplate(string(templBytes), templFP)
}

// formatCode takes a string representing golang code and attempts to return a
// formated copy of that code.  If formatting fails, a warning is logged and
// the original code is returned.
func formatCode(code []byte) []byte {
	formatted, err := format.Source(code)

	if err != nil {
		log.WithError(err).Warn("Code formatting error, generated service will not build, outputting unformatted code")
		// return code so at least we get something to examine
		return code
	}

	return formatted
}
