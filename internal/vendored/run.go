package vendored

import (
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"al.essio.dev/pkg/shellescape"
)

// Command runs exec.Command assuming name is one of the vendored binaries
// it also prints the command in copypastable form
func Command(name string, args ...string) *exec.Cmd {
	vendoredName := filepath.Join(".bin", name)

	quotedArgs := make([]string, len(args))
	for i, arg := range args {
		quotedArgs[i] = shellescape.Quote(arg)
	}
	log.Printf("Running command: \n%s %s\n", vendoredName, strings.Join(quotedArgs, " "))

	return exec.Command(vendoredName, args...)
}
