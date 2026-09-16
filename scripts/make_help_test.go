package scripts

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMakeHelpListsDocumentedTargets(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate this test file")
	}

	command := exec.Command("make", "help")
	command.Dir = filepath.Dir(filepath.Dir(sourceFile))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make help failed: %v\n%s", err, output)
	}

	help := string(output)
	if !strings.Contains(help, "Usage:   make <target>") {
		t.Fatalf("make help did not print usage:\n%s", help)
	}
	if !strings.Contains(help, "List documented Makefile targets.") {
		t.Fatalf("make help did not list the help target:\n%s", help)
	}
}
