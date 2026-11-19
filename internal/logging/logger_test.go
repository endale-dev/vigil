package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewFormatsByEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		jsonOutput  bool
	}{
		{name: "production", environment: "production", jsonOutput: true},
		{name: "development", environment: "development"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := New(tt.environment, &output)
			logger.Info("server ready", "address", ":8080")
			got := strings.TrimSpace(output.String())
			if tt.jsonOutput {
				var record map[string]any
				if err := json.Unmarshal([]byte(got), &record); err != nil {
					t.Fatalf("expected JSON log record: %v", err)
				}
				if record["msg"] != "server ready" || record["address"] != ":8080" {
					t.Fatalf("unexpected JSON log record: %#v", record)
				}
				return
			}
			if !strings.Contains(got, "msg=\"server ready\"") || !strings.Contains(got, "address=:8080") {
				t.Fatalf("unexpected text log record: %s", got)
			}
		})
	}
}
