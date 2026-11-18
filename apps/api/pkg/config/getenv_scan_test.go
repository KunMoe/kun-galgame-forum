package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The one-off tools under cmd/ never report a snapshot, so they are outside
// this rule; internal/testdb reads the test harness's variables.
var serviceRoots = []string{"../../internal", "../../cmd/server", "../../cmd/migrate"}

var directRead = regexp.MustCompile(`os\.(Getenv|LookupEnv)\(`)

func TestServicesReadEnvironmentThroughConfig(t *testing.T) {
	for _, root := range serviceRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == "testdb" {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if loc := directRead.FindIndex(src); loc != nil {
				line := 1 + strings.Count(string(src[:loc[0]]), "\n")
				t.Errorf("%s:%d reads the environment directly; use config.Env so the console sees it", path, line)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
