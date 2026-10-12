package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPrintJSON(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "map is printed as indented JSON",
			value: map[string]interface{}{"balance": json.Number("1.5"), "address": "T..."},
			want:  "{\n  \"address\": \"T...\",\n  \"balance\": 1.5\n}\n",
		},
		{
			name:    "infinite float fails instead of printing nothing",
			value:   map[string]interface{}{"amount": math.Inf(1)},
			wantErr: true,
		},
		{
			name:    "invalid json.Number fails instead of printing nothing",
			value:   map[string]interface{}{"amount": json.Number("1.2.3")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&out)

			err := printJSON(cmd, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("printJSON() error = nil, want an encoding error")
				}
				if !strings.Contains(err.Error(), "encode result as JSON") {
					t.Errorf("printJSON() error = %q, want it to name the encoding step", err)
				}
				if out.Len() != 0 {
					t.Errorf("printJSON() wrote %q on error, want no output", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("printJSON() error = %v", err)
			}
			if got := out.String(); got != tt.want {
				t.Errorf("printJSON() wrote %q, want %q", got, tt.want)
			}
		})
	}
}

// A command whose result cannot be encoded must fail, so tronctl exits
// non-zero instead of printing an empty line and reporting success.
func TestPrintJSON_CommandFails(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{
		Use: "broken",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printJSON(cmd, map[string]interface{}{"amount": math.NaN()})
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(&out)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute() error = nil, want the encoding error")
	}
	if out.Len() != 0 {
		t.Errorf("command wrote %q, want no output", out.String())
	}
}

// errWriter fails every write, like a closed pipe.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

func TestPrintJSON_WriteError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errWriter{})
	err := printJSON(cmd, map[string]interface{}{"ok": true})
	if err == nil || !strings.Contains(err.Error(), "broken pipe") {
		t.Fatalf("printJSON() error = %v, want the write error", err)
	}
}

func TestPrintJSON_ErrorUnwraps(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := printJSON(cmd, map[string]interface{}{"amount": math.Inf(-1)})
	var unsupported *json.UnsupportedValueError
	if !errors.As(err, &unsupported) {
		t.Fatalf("printJSON() error = %v, want it to wrap *json.UnsupportedValueError", err)
	}
}
