package utils

import "strings"

// goKitTemplateVersion returns the semantic version of go-kit as found in the
// template file path.
func GoKitTemplateFPVersion(templFP string) string {
	return strings.Split(templFP, "/")[0]
}

// TemplatePathToActual accepts a templateFilePath and the svcName of the
// service and returns what the relative file path of what should be written to
// disk
func TemplatePathToActual(templFilePath, svcName string) string {
	// Switch "NAME" in path with svcName.
	// i.e. for svcName = addsvc; /NAME-server -> /addsvc-service/addsvc-server
	actual := strings.Replace(templFilePath, "NAME", svcName, -1)
	actual = strings.TrimSuffix(actual, "template")

	// Remove the template version sub-directory path
	actual = strings.TrimPrefix(actual, GoKitTemplateFPVersion(actual)+"/")

	// Hoist templates up from service named directory.
	actual = strings.TrimPrefix(actual, svcName+"-service/")

	return actual
}
