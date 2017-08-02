package utils

import "testing"

const KitCompatVersion = "bbb2306"
const Kit030 = "v0.3.0"
const Kit040 = "v0.4.0"
const Kit050 = "v0.5.0"

func TestGoKitTemplateFPVersion(t *testing.T) {
	pathToWants := map[string]string{
		KitCompatVersion + "/NAME-service/":                KitCompatVersion,
		KitCompatVersion + "/NAME-service/test.gotemplate": KitCompatVersion,
		KitCompatVersion + "/NAME-service/NAME-server":     KitCompatVersion,
		Kit030 + "/NAME-service/":                          Kit030,
		Kit030 + "/NAME-service/test.gotemplate":           Kit030,
		Kit030 + "/NAME-service/NAME-server":               Kit030,
		Kit040 + "/NAME-service/":                          Kit040,
		Kit040 + "/NAME-service/test.gotemplate":           Kit040,
		Kit040 + "/NAME-service/NAME-server":               Kit040,
		Kit050 + "/NAME-service/":                          Kit050,
		Kit050 + "/NAME-service/test.gotemplate":           Kit050,
		Kit050 + "/NAME-service/NAME-server":               Kit050,
	}

	for path, want := range pathToWants {
		if got := GoKitTemplateFPVersion(path); got != want {
			t.Fatalf("\n`%v` got\n`%v` wanted", got, want)
		}
	}

}

func TestTemplatePathToActual(t *testing.T) {
	pathToWants := map[string]string{
		KitCompatVersion + "/NAME-service/":                "",
		KitCompatVersion + "/NAME-service/test.gotemplate": "test.go",
		KitCompatVersion + "/NAME-service/NAME-server":     "package-server",
		Kit030 + "/NAME-service/":                          "",
		Kit030 + "/NAME-service/test.gotemplate":           "test.go",
		Kit030 + "/NAME-service/NAME-server":               "package-server",
		Kit040 + "/NAME-service/":                          "",
		Kit040 + "/NAME-service/test.gotemplate":           "test.go",
		Kit040 + "/NAME-service/NAME-server":               "package-server",
		Kit050 + "/NAME-service/":                          "",
		Kit050 + "/NAME-service/test.gotemplate":           "test.go",
		Kit050 + "/NAME-service/NAME-server":               "package-server",
	}

	for path, want := range pathToWants {
		if got := TemplatePathToActual(path, "package"); got != want {
			t.Fatalf("\n`%v` got\n`%v` wanted", got, want)
		}
	}
}
