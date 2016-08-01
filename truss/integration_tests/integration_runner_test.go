// +build integration

package integration

import (
	"testing"
)

func TestTruss(t *testing.T) {
	allPassed := runIntegrationTests()
	if !allPassed {
		t.Fail()
	}
}
