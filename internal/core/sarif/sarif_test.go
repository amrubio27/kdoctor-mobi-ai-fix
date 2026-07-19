package sarif

import (
	"reflect"
	"strings"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestParseGolden(t *testing.T) {
	got, err := Parse(strings.NewReader(`{
		"version": "2.1.0",
		"runs": [{
			"tool": { "driver": { "name": "detekt" } },
			"results": [{
				"ruleId": "style:MagicNumber",
				"level": "warning",
				"message": { "text": "Magic number" },
				"locations": [{
					"physicalLocation": {
						"artifactLocation": { "uri": "src/Foo.kt" },
						"region": { "startLine": 12, "startColumn": 5 }
					}
				}]
			}]
		}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d findings", len(got))
	}
	want := types.Finding{
		Rule: "style:MagicNumber", Severity: types.SeverityWarning,
		File: "src/Foo.kt", Line: 12, Column: 5, Message: "Magic number",
	}
	if !reflect.DeepEqual(got[0], want) {
		t.Fatalf("\nwant %+v\ngot  %+v", want, got[0])
	}
}

func TestParseRejectsOldVersion(t *testing.T) {
	_, err := Parse(strings.NewReader(`{"version":"0.0.0","runs":[]}`))
	if err == nil {
		t.Fatal("expected error on unsupported version")
	}
	if !strings.Contains(err.Error(), "0.0.0") {
		t.Fatalf("error should reference version, got %q", err.Error())
	}
}

func TestMapSARIFLevel(t *testing.T) {
	cases := map[string]types.Severity{
		"error":   types.SeverityError,
		"warning": types.SeverityWarning,
		"note":    types.SeverityInfo,
		"":        "",
		"none":    "",
		"random":  "",
	}
	for input, want := range cases {
		if got := MapSARIFLevel(input); got != want {
			t.Fatalf("MapSARIFLevel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseMultipleResults(t *testing.T) {
	got, err := Parse(strings.NewReader(`{
		"version": "2.1.0",
		"runs": [{
			"tool": { "driver": { "name": "detekt" } },
			"results": [
				{"ruleId": "A", "message": {"text": "a"}},
				{"ruleId": "B", "level": "error", "message": {"text": "b"}}
			]
		}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Severity != "" {
		t.Fatalf("A sin level debe quedar con severity vacío, got %q", got[0].Severity)
	}
	if got[1].Severity != types.SeverityError {
		t.Fatalf("B con level=error debe ser error, got %q", got[1].Severity)
	}
}
