package getstarted

import (
	"bytes"
	"os"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	gogen "github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"
)

type protoInfo struct {
	alias string
}

func (p protoInfo) FileName() string {
	return p.PackageName() + ".proto"
}

func (p protoInfo) PackageName() string {
	a := p.alias
	a = strings.Replace(a, "-", "", -1)
	a = strings.Replace(a, " ", "", -1)

	a = strings.ToLower(a)
	return a
}

func (p protoInfo) ServiceName() string {
	a := p.alias
	a = strings.Replace(a, "-", "_", -1)
	a = strings.Replace(a, " ", "_", -1)
	return gogen.CamelCase(a)
}

// Do writes a default protobuf file to the current directory, in a file named
// based on the 'pkg' param, defaulting to "getstarted.proto" if pkg is empty.
// If the file exists, Do prints a warning and returns a non-zero exit code.
// The non-zero exit code is to enable using the return from this function in
// os.Exit().
func Do(pkg string) int {
	const fallbackFName = "get_started"
	if pkg == "" {
		pkg = fallbackFName
	}
	pkg = removeDotProtoSuffix(pkg)
	pinfo := protoInfo{
		alias: pkg,
	}
	if _, err := os.Stat(pinfo.FileName()); err == nil {
		existingFile, err := renderTemplate("existingFileMsg", existingFileMsg, pinfo)
		if err != nil {
			log.Error(err)
			return 1
		}
		log.Error(string(existingFile))
		return 1
	}
	f, err := os.Create(pinfo.FileName())
	if err != nil {
		log.Error(errors.Wrapf(err, "cannot create %q", pinfo.FileName()))
		return 1
	}

	code, err := renderTemplate(pinfo.FileName(), starterProto, pinfo)
	if err != nil {
		log.Error(err)
		return 1
	}

	_, err = f.Write(code)
	if err != nil {
		log.Error(errors.Wrapf(err, "cannot write default contents to %q", pinfo.FileName()))
		return 1
	}
	nextStep, err := renderTemplate("nextStepMsg", nextStepMsg, pinfo)
	if err != nil {
		log.Error(err)
		return 1
	}
	log.Info(string(nextStep))
	return 0
}

// removeDotProtoSuffix exists to preempt and warn a user who enters a name
// containing `.proto`. It will warn the user of their incorrect input and will
// demonstrate how their input can be corrected. Then, the program continues
// using the corrected input it warned about.
func removeDotProtoSuffix(pkg string) string {
	const dotProtoInName = `The name you provided has a suffix of '.proto' when it should not. Instead of
'{{.Got}}', you should provide '{{.Want}}'. Here's an example of the correct
command to enter next time:

	truss --getstarted {{.Want}}

For now this program is continuing as though you used '{{.Want}}'.
`
	want := strings.Replace(pkg, ".proto", "", -1)
	if strings.HasSuffix(pkg, ".proto") {
		executor := struct{ Got, Want string }{pkg, want}
		warn, err := renderTemplate("dotProtoInName", dotProtoInName, executor)
		if err != nil {
			log.Error(err)
		}
		log.Warn(string(warn))
	}
	return want
}

func renderTemplate(name string, tmpl string, executor interface{}) ([]byte, error) {
	codeTemplate := template.Must(template.New(name).Parse(tmpl))

	code := bytes.NewBuffer(nil)
	err := codeTemplate.Execute(code, executor)
	if err != nil {
		return nil, errors.Wrapf(err, "attempting to execute template %q", name)
	}
	return code.Bytes(), nil
}
