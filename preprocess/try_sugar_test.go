package preprocess

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandTrySugar_BareIdentInline(t *testing.T) {
	// "try EXPR or VARIABLE" with no body+end following should be
	// treated as inline form (variable is the fallback value),
	// NOT as block form (variable is error binding).
	tests := []struct {
		name   string
		input  string
		expect string // substring that must appear in expanded output
		reject string // substring that must NOT appear
	}{
		{
			name:   "bare variable as fallback",
			input:  "x = \"fallback\"\nwk = try 1/0 or x\nputs(wk)\n",
			expect: "or _err",  // should be expanded to inline form with _err binding
			reject: "\nor x\n", // should NOT keep x as error binding
		},
		{
			name:   "block form with body+end is preserved",
			input:  "result = try conv.to_i(\"bad\") or err\n  42\nend\n",
			reject: "or _err", // should NOT expand — it's block form
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ExpandTrySugar(tt.input)
			if tt.expect != "" {
				assert.Contains(t, got, tt.expect,
					"expanded output should contain %q\ngot:\n%s", tt.expect, got)
			}
			if tt.reject != "" {
				assert.NotContains(t, got, tt.reject,
					"expanded output should NOT contain %q\ngot:\n%s", tt.reject, got)
			}
		})
	}
}

func TestExpandTrySugar_BareIdentCompiles(t *testing.T) {
	// End-to-end: the expanded source should have "end" closing the
	// try block when a bare variable is used as inline fallback.
	input := "x = \"fallback\"\nwk = try 1/0 or x\nputs(wk)\n"
	got, _ := ExpandTrySugar(input)
	lines := strings.Split(got, "\n")
	hasEnd := false
	for _, l := range lines {
		if strings.TrimSpace(l) == "end" {
			hasEnd = true
			break
		}
	}
	assert.True(t, hasEnd, "inline try-or-variable must expand with closing 'end'\ngot:\n%s", got)
}
