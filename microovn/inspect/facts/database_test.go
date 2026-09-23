package facts

import (
	"strings"
	"testing"
)

func TestParseDBCfg(t *testing.T) {
	for _, database := range []string{"NB", "SB"} {
		t.Run(database+" valid", func(t *testing.T) {
			value, err := parseDBCfg(database, " 42\n")
			if err != nil {
				t.Fatalf("parseDBCfg returned an error: %v", err)
			}
			if value != 42 {
				t.Fatalf("parseDBCfg returned %d, want 42", value)
			}
		})

		t.Run(database+" malformed", func(t *testing.T) {
			_, err := parseDBCfg(database, "not-a-number")
			if err == nil {
				t.Fatal("parseDBCfg accepted malformed output")
			}
			if !strings.Contains(err.Error(), database) {
				t.Fatalf("parseDBCfg error %q does not identify %s", err, database)
			}
		})
	}
}
