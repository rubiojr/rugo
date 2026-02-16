package gobridge

import _ "embed"

//go:embed helpers/rune.go
var runeHelperSrc string

//go:embed helpers/string_slice.go
var stringSliceHelperSrc string

//go:embed helpers/byte_slice.go
var byteSliceHelperSrc string

// stripBuildTag removes the //go:build ignore line and package declaration
// from embedded helper source, returning only the function bodies.
func stripBuildTag(src string) string {
	// Find the end of the package line and skip everything before it
	lines := ""
	inBody := false
	for _, line := range splitLines(src) {
		if !inBody {
			if len(line) > 0 && line[0] == 'f' { // starts with "func"
				inBody = true
			}
			if !inBody {
				continue
			}
		}
		lines += line + "\n"
	}
	return lines
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func helperFromFile(key, src string) RuntimeHelper {
	return RuntimeHelper{Key: key, Code: stripBuildTag(src)}
}
