package compiler

import (
	"fmt"
	"strings"
	"unicode"
)

// LineMap tracks the mapping from preprocessed line numbers to original source line numbers.
// Index is 0-based preprocessed line, value is 1-based original line.
type LineMap struct {
	mapping []int
}

// NewLineMap creates a 1:1 line map for n lines.
func NewLineMap(n int) *LineMap {
	m := make([]int, n)
	for i := range m {
		m[i] = i + 1
	}
	return &LineMap{mapping: m}
}

// Mapping returns the preprocessed→original line mapping slice.
func (lm *LineMap) Mapping() []int {
	return lm.mapping
}

var rugoKeywords = map[string]bool{
	"if": true, "elsif": true, "else": true, "end": true,
	"while": true, "for": true, "in": true, "def": true,
	"return": true, "require": true, "break": true, "next": true,
	"true": true, "false": true, "nil": true, "import": true,
	"test": true, "try": true, "or": true,
	"spawn": true, "parallel": true,
}

var rugoBuiltins = map[string]bool{
	"puts": true, "print": true,
	"len": true, "append": true,
}

// StripComments removes # comments from source, respecting string boundaries.
func StripComments(src string) string {
	var sb strings.Builder
	inString := false
	escaped := false
	for i := 0; i < len(src); i++ {
		ch := src[i]
		if escaped {
			sb.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			sb.WriteByte(ch)
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			sb.WriteByte(ch)
			continue
		}
		if ch == '#' && !inString {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				sb.WriteByte('\n')
			}
			continue
		}
		sb.WriteByte(ch)
	}
	return sb.String()
}

// Preprocess performs line-level transformations:
// 1. Parenthesis-free function calls: `puts "foo"` → `puts("foo")`
// 2. Shell fallback: unknown idents → `__shell__("cmd line")`
//
// It uses positional resolution at the top level: a function name is only
// recognized after its `def` line has been encountered. Inside function bodies,
// all function names (allFuncs) are visible to allow forward references.
//
// Returns the preprocessed source and a line map (preprocessed line 0-indexed
// → original line 1-indexed). If lineMap is nil, the mapping is 1:1.
func Preprocess(src string, allFuncs map[string]bool) (string, []int, error) {
	// Desugar compound assignment operators before other transformations.
	src = expandCompoundAssign(src)

	// Expand backtick expressions before try sugar (backticks may appear inside try).
	src = expandBackticks(src)

	// Expand single-line try forms into block form before line processing.
	var tryLineMap []int
	src, tryLineMap = expandTrySugar(src)

	// Expand single-line spawn forms into block form.
	src, tryLineMap = expandSpawnSugar(src, tryLineMap)

	lines := strings.Split(src, "\n")
	var result []string

	topLevelFuncs := make(map[string]bool)
	var blockStack []string // tracks "def", "if", "while"
	defDepth := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		firstToken, _ := scanFirstToken(trimmed)

		// Choose func set: inside def bodies use allFuncs (forward refs),
		// at top level use only functions defined above this point.
		var funcs map[string]bool
		if defDepth > 0 {
			funcs = allFuncs
		} else {
			funcs = topLevelFuncs
		}

		processed := preprocessLine(line, funcs)
		// Detect orphan "or" on shell fallback lines
		if strings.Contains(processed, `__shell__("`) {
			if hasOrphanOr(trimmed) {
				origLine := i + 1
				if tryLineMap != nil && i < len(tryLineMap) {
					origLine = tryLineMap[i]
				}
				return "", nil, fmt.Errorf("line %d: `or` without `try` — did you mean `try %s`?", origLine, trimmed)
			}
		}
		result = append(result, processed)

		// Update block tracking after preprocessing the line.
		switch firstToken {
		case "def":
			blockStack = append(blockStack, "def")
			defDepth++
			// Register the function name so subsequent top-level lines see it.
			rest := strings.TrimSpace(trimmed[4:])
			name, _ := scanFirstToken(rest)
			if isIdent(name) {
				topLevelFuncs[name] = true
			}
		case "test":
			blockStack = append(blockStack, "test")
		case "if":
			blockStack = append(blockStack, "if")
		case "while":
			blockStack = append(blockStack, "while")
		case "for":
			blockStack = append(blockStack, "for")
		case "try":
			blockStack = append(blockStack, "try")
		case "spawn":
			blockStack = append(blockStack, "spawn")
		case "parallel":
			blockStack = append(blockStack, "parallel")
		case "end":
			if len(blockStack) > 0 {
				top := blockStack[len(blockStack)-1]
				blockStack = blockStack[:len(blockStack)-1]
				if top == "def" {
					defDepth--
				}
			}
		}
	}
	return strings.Join(result, "\n"), tryLineMap, nil
}

func preprocessLine(line string, userFuncs map[string]bool) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line
	}

	// Extract leading whitespace
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	// Rewrite `test.func(...)` to `__tmod__.func(...)` anywhere in the line,
	// since `test` is both a keyword (for test blocks) and a module name.
	// Only rewrite outside of string literals.
	line = rewriteTestModule(line)
	trimmed = strings.TrimSpace(line)

	// Check if line is assignment: `ident = ...` — leave alone
	// Check if line starts with keyword — leave alone
	// Check if line has parens on first call — leave alone

	firstToken, rest := scanFirstToken(trimmed)
	if firstToken == "" {
		return line
	}

	// Keywords — never touch
	if rugoKeywords[firstToken] {
		return line
	}

	// If first token is not an identifier, leave alone (e.g. starts with number, string, bracket)
	if !isIdent(firstToken) {
		return line
	}

	// Check what follows the first token
	restTrimmed := strings.TrimSpace(rest)

	// Assignment: `x = ...` — leave alone
	if len(restTrimmed) > 0 && restTrimmed[0] == '=' && (len(restTrimmed) < 2 || restTrimmed[1] != '=') {
		return line
	}

	// Already has parens: `foo(...)` — leave alone
	if len(restTrimmed) > 0 && restTrimmed[0] == '(' {
		return line
	}

	// Dot access: `ns.func(...)` — leave alone (handled by parser)
	if len(restTrimmed) > 0 && restTrimmed[0] == '.' {
		return line
	}

	// Index access: `arr[0]` — leave alone
	if len(restTrimmed) > 0 && restTrimmed[0] == '[' {
		return line
	}

	// Operator follows: `x + y`, `x == y` etc — leave alone (it's an expression)
	// But only if the first token could be a variable (known func/builtin or we
	// can't tell), not an unknown command like `ls -la`
	if len(restTrimmed) > 0 && isOperatorStart(restTrimmed[0]) {
		if rugoBuiltins[firstToken] || userFuncs[firstToken] {
			return line
		}
		// Unknown ident followed by operator — it's a shell command
		// e.g. `ls -la`, `uname -a`
		return indent + `__shell__("` + shellEscape(trimmed) + `")`
	}

	// Empty rest — bare ident. If it's a known function/builtin, it's a no-arg call.
	// Otherwise it's a shell command.
	if restTrimmed == "" {
		if rugoBuiltins[firstToken] || userFuncs[firstToken] {
			return indent + firstToken + "()"
		}
		// Shell: single command like `ls`
		return indent + `__shell__("` + shellEscape(firstToken) + `")`
	}

	// If the ident is a known builtin or user function, it's a paren-free call
	if rugoBuiltins[firstToken] || userFuncs[firstToken] {
		// Rewrite `func arg1, arg2` → `func(arg1, arg2)`
		return indent + firstToken + "(" + restTrimmed + ")"
	}

	// Otherwise it's a shell command — the whole line is the command
	return indent + `__shell__("` + shellEscape(trimmed) + `")`
}

// hasOrphanOr detects ` or ` used as a Rugo fallback keyword in a line
// that has no `try` prefix. This catches mistakes like:
//
//	timeout 30 ping host or "fallback"
//
// which should be:
//
//	try timeout 30 ping host or "fallback"
func hasOrphanOr(line string) bool {
	inStr := false
	for i := 0; i < len(line); i++ {
		if line[i] == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		// Match " or " or " or" at end of line, as a word boundary
		if line[i] == ' ' && i+3 <= len(line) && line[i+1:i+3] == "or" {
			after := i + 3
			if after == len(line) || line[after] == ' ' || line[after] == '\t' {
				return true
			}
		}
	}
	return false
}

// scanFirstToken extracts the first whitespace-delimited token and the rest.
func scanFirstToken(s string) (string, string) {
	i := 0
	for i < len(s) && !unicode.IsSpace(rune(s[i])) && s[i] != '(' && s[i] != '[' && s[i] != '=' {
		i++
	}
	if i == 0 {
		return "", s
	}
	return s[:i], s[i:]
}

func isIdent(s string) bool {
	if len(s) == 0 {
		return false
	}
	ch := rune(s[0])
	if !unicode.IsLetter(ch) && ch != '_' {
		return false
	}
	for _, c := range s[1:] {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
			return false
		}
	}
	return true
}

func isOperatorStart(ch byte) bool {
	switch ch {
	case '+', '-', '*', '/', '%', '<', '>', '!', '&', '|':
		return true
	}
	return false
}

// shellEscape escapes a string for embedding in a Go/rugo string literal.
func shellEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// ScanFuncDefs does a quick scan to find all `def name(` patterns
// so the preprocessor knows which identifiers are user functions.
func ScanFuncDefs(src string) map[string]bool {
	funcs := make(map[string]bool)
	lines := strings.Split(src, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") {
			rest := strings.TrimSpace(trimmed[4:])
			name, _ := scanFirstToken(rest)
			if isIdent(name) {
				funcs[name] = true
			}
		}
	}
	return funcs
}

// ProcessInterpolation converts "Hello #{expr}" to format string + args.
// Returns the format string and a list of expression strings.
func ProcessInterpolation(s string) (format string, exprs []string) {
	var fmt strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '#' && s[i+1] == '{' {
			// Find matching }
			depth := 1
			j := i + 2
			for j < len(s) && depth > 0 {
				if s[j] == '{' {
					depth++
				} else if s[j] == '}' {
					depth--
				}
				j++
			}
			expr := s[i+2 : j-1]
			exprs = append(exprs, expr)
			fmt.WriteString("%v")
			i = j
		} else {
			fmt.WriteByte(s[i])
			i++
		}
	}
	return fmt.String(), exprs
}

// HasInterpolation checks if a string contains #{} interpolation.
func HasInterpolation(s string) bool {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '#' && s[i+1] == '{' {
			return true
		}
	}
	return false
}

// rewriteTestModule replaces `test.` with `__tmod__.` outside of string literals.
// This resolves the conflict between `test` as a keyword and as a module name.
func rewriteTestModule(line string) string {
	var sb strings.Builder
	inString := false
	escaped := false
	i := 0
	for i < len(line) {
		ch := line[i]
		if escaped {
			sb.WriteByte(ch)
			escaped = false
			i++
			continue
		}
		if ch == '\\' && inString {
			sb.WriteByte(ch)
			escaped = true
			i++
			continue
		}
		if ch == '"' {
			inString = !inString
			sb.WriteByte(ch)
			i++
			continue
		}
		if !inString && i+5 <= len(line) && line[i:i+5] == "test." {
			// Make sure it's a word boundary (not part of a larger identifier)
			if i == 0 || !isIdentChar(line[i-1]) {
				sb.WriteString("__tmod__.")
				i += 5
				continue
			}
		}
		sb.WriteByte(ch)
		i++
	}
	return sb.String()
}

func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// expandTrySugar expands single-line try forms into the full block form.
// Returns the expanded source and a mapping from output line (0-indexed) to
// original input line (1-indexed).
//
//	try EXPR            → try EXPR or _ nil end
//	try EXPR or DEFAULT → try EXPR or _ DEFAULT end
//	x = try EXPR ...    → x = try EXPR ... (same, in assignment context)
//
// Multi-line try blocks (try ... or ident ... end) are left untouched.
func expandTrySugar(src string) (string, []int) {
	lines := strings.Split(src, "\n")
	var result []string
	var lineMap []int
	for i, line := range lines {
		origLine := i + 1
		trimmed := strings.TrimSpace(line)
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		// Extract assignment prefix if present: "x = try ..." → prefix="x = ", tryPart="try ..."
		prefix := ""
		tryPart := trimmed
		if eqIdx := strings.Index(trimmed, "= try "); eqIdx >= 0 {
			// Verify it's not == (comparison)
			if eqIdx == 0 || trimmed[eqIdx-1] != '=' && trimmed[eqIdx-1] != '!' && trimmed[eqIdx-1] != '<' && trimmed[eqIdx-1] != '>' {
				prefix = trimmed[:eqIdx+2]
				tryPart = strings.TrimSpace(trimmed[eqIdx+2:])
			}
		}

		if !strings.HasPrefix(tryPart, "try ") {
			result = append(result, line)
			lineMap = append(lineMap, origLine)
			continue
		}

		rest := strings.TrimSpace(tryPart[4:])

		// Skip expansion when the expression is a block keyword (spawn, parallel)
		// that spans multiple lines — let the parser handle it.
		firstTok, _ := scanFirstToken(rest)
		if firstTok == "spawn" || firstTok == "parallel" {
			result = append(result, line)
			lineMap = append(lineMap, origLine)
			continue
		}

		orIdx := findTopLevelOr(rest)
		if orIdx >= 0 {
			// Check if what follows "or" is just an identifier (error variable) — this
			// is the block form "try EXPR or ident\n BODY\n end", leave it untouched.
			afterOr := strings.TrimSpace(rest[orIdx+2:])
			afterOrTok, afterOrRest := scanFirstToken(afterOr)
			if isIdent(afterOrTok) && !rugoKeywords[afterOrTok] && strings.TrimSpace(afterOrRest) == "" {
				// Split the expression onto its own line so preprocessLine can
				// apply shell fallback to bare identifiers inside try.
				expr := strings.TrimSpace(rest[:orIdx])
				result = append(result, indent+prefix+"try")
				lineMap = append(lineMap, origLine)
				result = append(result, indent+"  "+expr)
				lineMap = append(lineMap, origLine)
				result = append(result, indent+"or "+afterOrTok)
				lineMap = append(lineMap, origLine)
				continue
			}

			// "try EXPR or DEFAULT" → expand to block form
			// Put the expression on its own line so preprocessLine can apply shell fallback.
			expr := strings.TrimSpace(rest[:orIdx])
			dflt := strings.TrimSpace(rest[orIdx+2:])
			if dflt == "" {
				dflt = "nil"
			}
			result = append(result, indent+prefix+"try")
			lineMap = append(lineMap, origLine)
			result = append(result, indent+"  "+expr)
			lineMap = append(lineMap, origLine)
			result = append(result, indent+"or _err")
			lineMap = append(lineMap, origLine)
			result = append(result, indent+"  "+dflt)
			lineMap = append(lineMap, origLine)
			result = append(result, indent+"end")
			lineMap = append(lineMap, origLine)
		} else {
			// "try EXPR" with no "or" → silent recovery (nil on failure)
			result = append(result, indent+prefix+"try")
			lineMap = append(lineMap, origLine)
			result = append(result, indent+"  "+rest)
			lineMap = append(lineMap, origLine)
			result = append(result, indent+"or _err")
			lineMap = append(lineMap, origLine)
			result = append(result, indent+"  nil")
			lineMap = append(lineMap, origLine)
			result = append(result, indent+"end")
			lineMap = append(lineMap, origLine)
		}
	}
	return strings.Join(result, "\n"), lineMap
}

// expandSpawnSugar expands one-liner "spawn EXPR" into block form.
// "spawn EXPR"       → "spawn\n  EXPR\nend"
// "x = spawn EXPR"   → "x = spawn\n  EXPR\nend"
func expandSpawnSugar(src string, lineMap []int) (string, []int) {
	lines := strings.Split(src, "\n")
	var result []string
	var newMap []int
	for i, line := range lines {
		origLine := lineMap[i]
		trimmed := strings.TrimSpace(line)
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		// Extract assignment prefix: "x = spawn ..." → prefix="x = ", spawnPart="spawn ..."
		prefix := ""
		spawnPart := trimmed
		if eqIdx := strings.Index(trimmed, "= spawn "); eqIdx >= 0 {
			if eqIdx == 0 || trimmed[eqIdx-1] != '=' && trimmed[eqIdx-1] != '!' && trimmed[eqIdx-1] != '<' && trimmed[eqIdx-1] != '>' {
				prefix = trimmed[:eqIdx+2]
				spawnPart = strings.TrimSpace(trimmed[eqIdx+2:])
			}
		}

		if !strings.HasPrefix(spawnPart, "spawn ") && !strings.HasPrefix(spawnPart, "spawn(") {
			result = append(result, line)
			newMap = append(newMap, origLine)
			continue
		}

		rest := strings.TrimSpace(spawnPart[5:])
		// If rest is empty or starts with newline, it's already block form
		if rest == "" {
			result = append(result, line)
			newMap = append(newMap, origLine)
			continue
		}

		// One-liner: expand "spawn EXPR" to block form
		result = append(result, indent+prefix+"spawn")
		newMap = append(newMap, origLine)
		result = append(result, indent+"  "+rest)
		newMap = append(newMap, origLine)
		result = append(result, indent+"end")
		newMap = append(newMap, origLine)
	}
	return strings.Join(result, "\n"), newMap
}

// findTopLevelOr finds " or " at the top level (not inside parens, brackets, or strings).
// Returns the index of the start of " or " in s, or -1 if not found.
func findTopLevelOr(s string) int {
	depth := 0
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '(' || ch == '[' || ch == '{' {
			depth++
		} else if ch == ')' || ch == ']' || ch == '}' {
			depth--
		}
		if depth == 0 && i+3 < len(s) && s[i:i+4] == " or " {
			return i + 1 // return index of 'o' in "or"
		}
	}
	return -1
}

// expandCompoundAssign desugars compound assignment operators.
//
//	x += y       → x = x + y
//	arr[0] += y  → arr[0] = arr[0] + y
//
// Handles +=, -=, *=, /=, %=. Respects string boundaries.
func expandCompoundAssign(src string) string {
	ops := []string{"+=", "-=", "*=", "/=", "%="}
	lines := strings.Split(src, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, line)
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		expanded := false
		for _, op := range ops {
			idx := findCompoundOp(trimmed, op)
			if idx < 0 {
				continue
			}
			lhs := strings.TrimSpace(trimmed[:idx])
			rhs := strings.TrimSpace(trimmed[idx+len(op):])
			arithOp := string(op[0])
			result = append(result, indent+lhs+" = "+lhs+" "+arithOp+" "+rhs)
			expanded = true
			break
		}
		if !expanded {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// findCompoundOp finds a compound operator (e.g. "+=") at the top level of a line,
// not inside strings, parens, or brackets. Returns the index or -1.
func findCompoundOp(s string, op string) int {
	inString := false
	escaped := false
	depth := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '(' || ch == '[' || ch == '{' {
			depth++
		} else if ch == ')' || ch == ']' || ch == '}' {
			depth--
		}
		if depth == 0 && i+len(op) <= len(s) && s[i:i+len(op)] == op {
			// Make sure it's not inside a comparison like "!=" by checking
			// the operator is preceded by a space or bracket/ident char.
			return i
		}
	}
	return -1
}

// expandBackticks converts `cmd` expressions to __capture__("cmd") calls.
// Backticks inside string literals are left untouched.
func expandBackticks(src string) string {
	var sb strings.Builder
	inString := false
	escaped := false
	for i := 0; i < len(src); i++ {
		ch := src[i]
		if escaped {
			sb.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			sb.WriteByte(ch)
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			sb.WriteByte(ch)
			continue
		}
		if ch == '`' && !inString {
			// Find the closing backtick
			j := i + 1
			for j < len(src) && src[j] != '`' {
				j++
			}
			if j < len(src) {
				cmd := src[i+1 : j]
				sb.WriteString(`__capture__("` + shellEscape(cmd) + `")`)
				i = j
				continue
			}
		}
		sb.WriteByte(ch)
	}
	return sb.String()
}
