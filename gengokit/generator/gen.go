package generator

import (
	"bytes"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	"github.com/TuneLab/go-truss/gengokit"
	"github.com/TuneLab/go-truss/gengokit/handler"
	templFiles "github.com/TuneLab/go-truss/gengokit/template"

	"github.com/TuneLab/go-truss/svcdef"
	"github.com/TuneLab/go-truss/truss"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

// GenerateGokit returns a gokit service generated from a service definition (svcdef),
// the package to the root of the generated service goPackage, the package
// to the .pb.go service struct files (goPBPackage) and any prevously generated files.
func GenerateGokit(sd *svcdef.Svcdef, conf gengokit.Config) ([]truss.NamedReadWriter, error) {
	te, err := gengokit.NewTemplateExecutor(sd, conf)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create template executor")
	}

	fpm := make(map[string]io.Reader, len(conf.PreviousFiles))
	for _, f := range conf.PreviousFiles {
		fpm[f.Name()] = f
	}

	var codeGenFiles []truss.NamedReadWriter

	for _, templFP := range templFiles.AssetNames() {
		file, err := generateResponseFile(templFP, te, fpm)
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
// template path to render, a templateExecutor to apply to the template, and a
// map of paths to files for the previous generation. It returns a
// truss.NamedReadWriter representing the generated file
func generateResponseFile(templFP string, te *gengokit.TemplateExecutor, prevGenMap map[string]io.Reader) (truss.NamedReadWriter, error) {
	var genCode io.Reader
	var err error

	// Get the actual path to the file rather than the template file path
	actualFP := templatePathToActual(templFP, te.PackageName)

	// If we are rendering the server and or the client
	if templFP == "NAME-service/handlers/server/server_handler.gotemplate" ||
		templFP == "NAME-service/handlers/client/client_handler.gotemplate" {
		file := prevGenMap[actualFP]
		h, err := handler.New(te.Service, file)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot parse previous handler: %q", actualFP)
		}

		if genCode, err = h.Render(templFP, te); err != nil {
			return nil, errors.Wrap(err, "cannot render template")
		}
	}

	// if no code has been generated just apply the template
	if genCode == nil {
		if genCode, err = applyTemplateFromPath(templFP, te); err != nil {
			return nil, errors.Wrap(err, "cannot render template")
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
func applyTemplateFromPath(templFilePath string, executor *gengokit.TemplateExecutor) (io.Reader, error) {
	templBytes, err := templFiles.Asset(templFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find template file: %v", templFilePath)
	}

	return applyTemplate(templBytes, templFilePath, executor)
}

func applyTemplate(templBytes []byte, templName string, executor *gengokit.TemplateExecutor) (io.Reader, error) {
	templateString := string(templBytes)

	codeTemplate, err := template.New(templName).Funcs(executor.FuncMap).Parse(templateString)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create template")
	}

	outputBuffer := bytes.NewBuffer(nil)
	err = codeTemplate.Execute(outputBuffer, executor)
	if err != nil {
		return nil, errors.Wrap(err, "template error")
	}

	return outputBuffer, nil
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
