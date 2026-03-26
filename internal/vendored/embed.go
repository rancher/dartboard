package vendored

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DestinationDir is the directory where embedded binaries are extracted at runtime
const DestinationDir = ".bin"

// SourceDir is the directory where embedded binaries are at compile time
const SourceDir = "bin"

// sourceFS is the virtual filesystem where embedded binaries are at runtime
//
//go:embed bin/*
var sourceFS embed.FS

// ExtractBinaries extracts embedded binaries at runtime
func ExtractBinaries() error {

	err := os.Mkdir(DestinationDir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = fs.WalkDir(sourceFS, SourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip root
		if d.IsDir() {
			return nil
		}

		// skip existing
		destFile := filepath.Join(".bin", strings.TrimPrefix(path, SourceDir+"/"))
		_, err = os.Stat(destFile)
		if err == nil {
			return nil
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check if file %v exists: %v", destFile, err)
		}

		// actually read and write
		content, err := sourceFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read an embedded file %v: %v", path, err)
		}

		err = os.WriteFile(destFile, content, 0755)
		if err != nil {
			return fmt.Errorf("failed to write %v: %v", destFile, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to read embedded files: %v", err)
	}

	return nil
}
