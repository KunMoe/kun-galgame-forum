package apiv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielgtaylor/huma/v2"
)

type problemsFile struct {
	Codes   []problemType   `json:"codes"`
	Reasons []problemReason `json:"reasons"`
}

func MarshalOpenAPI(api huma.API) ([]byte, error) {
	raw, err := api.OpenAPI().MarshalJSON()
	if err != nil {
		return nil, err
	}
	published, err := publishSpec(raw)
	if err != nil {
		return nil, err
	}
	return indentJSON(published)
}

func MarshalProblems() ([]byte, error) {
	raw, err := json.Marshal(problemsFile{
		Codes:   ProblemTypes(),
		Reasons: ProblemReasons(),
	})
	if err != nil {
		return nil, err
	}
	return indentJSON(raw)
}

func WriteSpecFiles(dir string, api huma.API) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	spec, err := MarshalOpenAPI(api)
	if err != nil {
		return fmt.Errorf("openapi: %w", err)
	}
	problems, err := MarshalProblems()
	if err != nil {
		return fmt.Errorf("problems: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "kungal-v1.json"), spec, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "problems.json"), problems, 0o644)
}

func indentJSON(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
