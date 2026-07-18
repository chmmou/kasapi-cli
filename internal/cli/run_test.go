package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

// runWriteE renders its success line through the writeResult wrapper so
// write commands honour --output like the read commands: table stays
// the bare human line, json/yaml wrap it in a message object scripts
// can parse.
func TestWriteResultRendersPerFormat(t *testing.T) {
	t.Parallel()
	r := cli.WriteResult{Message: "updated mailing list L"}

	var table bytes.Buffer
	if err := cli.Render(&table, cli.FormatTable, r); err != nil {
		t.Fatalf("Render table: %v", err)
	}
	if got := table.String(); got != "updated mailing list L\n" {
		t.Errorf("table output = %q, want the bare success line", got)
	}

	var jsonBuf bytes.Buffer
	if err := cli.Render(&jsonBuf, cli.FormatJSON, r); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var got struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(jsonBuf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, jsonBuf.String())
	}
	if got.Message != "updated mailing list L" {
		t.Errorf("json message = %q, want the success line", got.Message)
	}

	var yamlBuf bytes.Buffer
	if err := cli.Render(&yamlBuf, cli.FormatYAML, r); err != nil {
		t.Fatalf("Render yaml: %v", err)
	}
	if !strings.Contains(yamlBuf.String(), "message: updated mailing list L") {
		t.Errorf("yaml output = %q, want a message field", yamlBuf.String())
	}
}
