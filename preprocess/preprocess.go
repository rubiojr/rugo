package preprocess

import (
	"fmt"
	"github.com/rubiojr/rugo/util"
	"math"
	"strings"
	"unicode"
)

var RugoKeywords = map[string]bool{
	"if": true, "elsif": true, "else": true, "end": true,
	"while": true, "for": true, "in": true, "def": true,
	"return": true, "require": true, "break": true, "next": true,
	"true": true, "false": true, "nil": true, "import": true, "use": true,
	"rats": true, "try": true, "or": true,
	"spawn": true, "parallel": true, "bench": true, "fn": true,
	"struct": true, "with": true, "sandbox": true, "do": true,
}

var rugoBuiltins = map[string]bool{
	"puts": true, "print": true,
	"len": true, "append": true,
	"raise": true, "type_of": true,
	"exit": true,
}

// stripComments removes # comments from source, respecting string and backtick boundaries.
// Returns an error if an unterminated string literal is found.
func StripComments(src string) (string, error) {
	var sb strings.Builder
	inDouble := false
	inSingle := false
	inBacktick := false
	escaped := false
	stringStartLine := 0
	lineNum := 1
	for i := 0; i < len(src); i++ {
		ch := src[i]
		if ch == '\n' {
			// Detect unterminated string: if we're inside a string at a newline,
			// it's unclosed (Rugo doesn't support multiline strings).
			if inDouble {
				return "", fmt.Errorf("%d: unterminated string literal (opened at line %d)", lineNum, stringStartLine)
			}
			if inSingle {
				return "", fmt.Errorf("%d: unterminated string literal (opened at line %d)", lineNum, stringStartLine)
			}
			lineNum++
		}
		if escaped {
			sb.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' && (inDouble || inSingle) {
			sb.WriteByte(ch)
			escaped = true
			continue
		}
		if ch == '"' && !inSingle && !inBacktick {
			if !inDouble {
				stringStartLine = lineNum
			}
			inDouble = !inDouble
			sb.WriteByte(ch)
			continue
		}
		if ch == '\'' && !inDouble && !inBacktick {
			if !inSingle {
				stringStartLine = lineNum
			}
			inSingle = !inSingle
			sb.WriteByte(ch)
			continue
		}
		if ch == '`' && !inDouble && !inSingle {
			inBacktick = !inBacktick
			sb.WriteByte(ch)
			continue
		}
		if ch == '#' && !inDouble && !inSingle && !inBacktick {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				sb.WriteByte('\n')
				lineNum++
			}
			continue
		}
		sb.WriteByte(ch)
	}
	if inDouble || inSingle {
		return "", fmt.Errorf("%d: unterminated string literal (opened at line %d)", lineNum, stringStartLine)
	}
	return sb.String(), nil
}

// preprocess performs line-level transformations:
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
	// Reject user-written semicolons — they are reserved for internal use
	// by the preprocessor to disambiguate array literals on new lines.
	if err := RejectSemicolons(src); err != nil {
		return "", nil, err
	}

	// Reject trailing commas in array and hash literals.
	if err := RejectTrailingCommas(src); err != nil {
		return "", nil, err
	}

	// Rewrite hash colon syntax before other transformations:
	//   {foo: "bar"}  →  {"foo" => "bar"}
	src, err := ExpandHashColonSyntax(src)
	if err != nil {
		return "", nil, err
	}

	// Desugar compound assignment operators before other transformations.
	src = ExpandCompoundAssign(src)

	// Desugar array destructuring: a, b, c = expr → temp + index assignments.
	src = expandDestructuring(src)

	// Normalize "def name" (no parens) to "def name()" so the parser sees
	// a consistent form. "def name(params)" is left unchanged.
	src = expandDefParens(src)

	// Expand postfix if: "STMT if COND" → "if COND\nSTMT\nend"
	src = expandPostfixIf(src)

	// Expand backtick expressions before try sugar (backticks may appear inside try).
	src, err = expandBackticks(src)
	if err != nil {
		return "", nil, err
	}

	// Expand do...end trailing blocks into fn()...end before other expansions.
	// e.g. `vbox do ... end` → `vbox(fn() ... end)`
	src, err = expandDoEnd(src)
	if err != nil {
		return "", nil, err
	}

	// Expand single-line try forms into block form before line processing.
	var tryLineMap []int
	src, tryLineMap = ExpandTrySugar(src)

	// Expand single-line spawn forms into block form.
	src, tryLineMap = expandSpawnSugar(src, tryLineMap)

	// Expand inline fn bodies so paren-free calls inside them get preprocessed.
	src, tryLineMap = expandInlineFn(src, tryLineMap)

	lines := strings.Split(src, "\n")
	var result []string

	topLevelFuncs := make(map[string]bool)
	knownVars := make(map[string]bool)
	var blockStack []string // tracks "def", "if", "while"
	defDepth := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		firstToken, rest := scanFirstToken(trimmed)

		// Track variable assignments: `x = ...`
		if isIdent(firstToken) {
			rt := strings.TrimSpace(rest)
			if len(rt) > 0 && rt[0] == '=' && (len(rt) < 2 || rt[1] != '=') {
				knownVars[firstToken] = true
			}
		}

		// Track def parameters: `def foo(a, b)` or `def foo(a, b = nil)`
		if firstToken == "def" {
			if lp := strings.Index(trimmed, "("); lp != -1 {
				if rp := strings.Index(trimmed[lp:], ")"); rp != -1 {
					params := trimmed[lp+1 : lp+rp]
					for _, p := range strings.Split(params, ",") {
						p = strings.TrimSpace(p)
						// Strip default value: "b = nil" → "b"
						if eqIdx := strings.Index(p, "="); eqIdx != -1 {
							p = strings.TrimSpace(p[:eqIdx])
						}
						if isIdent(p) {
							knownVars[p] = true
						}
					}
				}
			}
		}

		// Track for-loop variables: `for x in ...`, `for k, v in ...`
		if firstToken == "for" {
			forRest := strings.TrimSpace(trimmed[3:])
			if inIdx := strings.Index(forRest, " in "); inIdx != -1 {
				vars := forRest[:inIdx]
				for _, v := range strings.Split(vars, ",") {
					v = strings.TrimSpace(v)
					if isIdent(v) {
						knownVars[v] = true
					}
				}
			}
		}

		// Track fn (lambda) parameters: `fn(a, b)` or `fn(a, b = nil)`
		// Look for fn( anywhere on the line (it may appear in arrays, hashes,
		// function args, etc.) and track all parameter names found.
		if strings.Contains(trimmed, "fn(") {
			searchFrom := 0
			for {
				lp := strings.Index(trimmed[searchFrom:], "fn(")
				if lp < 0 {
					break
				}
				absLP := searchFrom + lp
				// Ensure "fn" is not part of a larger identifier
				if absLP > 0 && (isAlphaNum(trimmed[absLP-1]) || trimmed[absLP-1] == '_') {
					searchFrom = absLP + 3
					continue
				}
				start := absLP + 3
				if rp := strings.Index(trimmed[start:], ")"); rp != -1 {
					params := trimmed[start : start+rp]
					for _, p := range strings.Split(params, ",") {
						p = strings.TrimSpace(p)
						// Strip default value: "b = nil" → "b"
						if eqIdx := strings.Index(p, "="); eqIdx != -1 {
							p = strings.TrimSpace(p[:eqIdx])
						}
						if isIdent(p) {
							knownVars[p] = true
						}
					}
					searchFrom = start + rp + 1
				} else {
					break
				}
			}
		}

		// Choose func set: inside def bodies use allFuncs (forward refs),
		// at top level use only functions defined above this point.
		var funcs map[string]bool
		if defDepth > 0 {
			funcs = allFuncs
		} else {
			funcs = topLevelFuncs
		}

		// Expand pipes before normal line processing
		var pipeErr error
		line, pipeErr = expandPipeLine(line, funcs)
		if pipeErr != nil {
			origLine := i + 1
			if tryLineMap != nil && i < len(tryLineMap) {
				origLine = tryLineMap[i]
			}
			return "", nil, fmt.Errorf("line %d: %s", origLine, pipeErr.Error())
		}

		processed := preprocessLine(line, funcs, knownVars)
		// Detect orphan "or" on shell fallback lines
		if strings.Contains(processed, `__shell__("`) {
			if hasOrphanOr(trimmed) {
				origLine := i + 1
				if tryLineMap != nil && i < len(tryLineMap) {
					origLine = tryLineMap[i]
				}
				return "", nil, fmt.Errorf("line %d: `or` without `try` — did you mean `try %s`?", origLine, trimmed)
			}
			// Detect misspelled keywords/builtins
			if closest := closestKeywordOrBuiltin(firstToken); closest != "" {
				origLine := i + 1
				if tryLineMap != nil && i < len(tryLineMap) {
					origLine = tryLineMap[i]
				}
				return "", nil, fmt.Errorf("line %d: unknown keyword `%s` — did you mean `%s`?", origLine, firstToken, closest)
			}
		}
		result = append(result, processed)

		// Update block tracking after preprocessing the line.
		switch firstToken {
		case "def":
			blockStack = append(blockStack, "def")
			defDepth++
			// Register the function name so subsequent top-level lines see it.
			if len(trimmed) < 5 || strings.TrimSpace(trimmed[4:]) == "" {
				origLine := i + 1
				if tryLineMap != nil && i < len(tryLineMap) {
					origLine = tryLineMap[i]
				}
				return "", nil, fmt.Errorf("line %d: `def` requires a function name — e.g. `def my_function()`", origLine)
			}
			rest := strings.TrimSpace(trimmed[4:])
			name, _ := scanFirstToken(rest)
			if isIdent(name) {
				topLevelFuncs[name] = true
			}
		case "rats":
			blockStack = append(blockStack, "rats")
		case "bench":
			blockStack = append(blockStack, "bench")
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
		case "fn":
			blockStack = append(blockStack, "fn")
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
	joined := strings.Join(result, "\n")

	// Desugar bare append: append(x, ...) → x = append(x, ...)
	joined = ExpandBareAppend(joined)

	// Insert ';' after sandbox lines to disambiguate from the next statement.
	// Without this, `sandbox\nputs(...)` would make the parser try to match
	// `puts` as a SandboxPerm ident.
	joined = insertSandboxSeparators(joined)

	// Insert ';' before lines starting with '[' to disambiguate array literals
	// from index suffix operations. Without this, `expr\n[1, 2, 3]` would be
	// parsed as `expr[1, 2, 3]` (index) instead of two separate statements.
	joined = InsertArraySeparators(joined)

	// Insert ';' before lines starting with '-' followed by a digit to
	// disambiguate negative literals from binary subtraction across lines.
	// Without this, `expr\n-1` would be parsed as `expr - 1` (subtraction)
	// instead of two separate statements. (git-bug 7f2fe9b)
	joined = InsertNegativeLiteralSeparators(joined)

	return joined, tryLineMap, nil
}

func preprocessLine(line string, userFuncs map[string]bool, knownVars map[string]bool) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line
	}

	// Extract leading whitespace
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	// Check if line is assignment: `ident = ...` — leave alone
	// Check if line starts with keyword — leave alone
	// Check if line has parens on first call — leave alone

	firstToken, rest := scanFirstToken(trimmed)
	if firstToken == "" {
		return line
	}

	// Keywords — never touch
	if RugoKeywords[firstToken] {
		return line
	}

	// If first token is not an identifier, check for dotted ident (module.func paren-free call)
	if !isIdent(firstToken) {
		if isDottedIdent(firstToken) {
			parts := strings.SplitN(firstToken, ".", 2)
			// If the object part is a known variable, this is field access (e.g. u.name),
			// not a paren-free namespace call — leave it alone.
			if knownVars[parts[0]] {
				return line
			}
			rt := strings.TrimSpace(rest)
			if rt == "" {
				// Bare dotted ident: `cli.run` → `cli.run()`
				return indent + firstToken + "()"
			}
			if rt[0] != '(' && rt[0] != '=' && !isOperatorStart(rt[0]) {
				return indent + firstToken + "(" + rt + ")"
			}
		}
		// Hyphenated command: `docker-compose up`, `apt-get install`, etc.
		// Hyphens are invalid in Rugo identifiers, so this is always a shell command.
		if isHyphenatedCommand(firstToken) {
			return indent + `__shell__("` + shellEscape(trimmed) + `")`
		}
		// Path-based command: `./script`, `../foo`, `/usr/bin/ls`
		// Paths are invalid in Rugo identifiers, so this is always a shell command.
		if isPathCommand(firstToken) {
			return indent + `__shell__("` + shellEscape(trimmed) + `")`
		}
		return line
	}

	// Check what follows the first token
	restTrimmed := strings.TrimSpace(rest)

	// Assignment: `x = ...` — check if RHS is a shell command
	if len(restTrimmed) > 0 && restTrimmed[0] == '=' && (len(restTrimmed) < 2 || restTrimmed[1] != '=') {
		rhs := strings.TrimSpace(restTrimmed[1:])
		rhsTok, rhsRest := scanFirstToken(rhs)
		if rhsTok != "" && isIdent(rhsTok) && !RugoKeywords[rhsTok] && !rugoBuiltins[rhsTok] && !userFuncs[rhsTok] && !knownVars[rhsTok] {
			rhsRestTrimmed := strings.TrimSpace(rhsRest)
			// Unknown ident with flag-like args: `ls -la`, `uname -a`
			if len(rhsRestTrimmed) > 0 && isOperatorStart(rhsRestTrimmed[0]) {
				return indent + firstToken + " = " + `__shell__("` + shellEscape(rhs) + `")`
			}
			// Unknown ident with non-expression args: `echo hello`
			if len(rhsRestTrimmed) > 0 && rhsRestTrimmed[0] != '(' && rhsRestTrimmed[0] != '[' && rhsRestTrimmed[0] != '.' {
				return indent + firstToken + " = " + `__shell__("` + shellEscape(rhs) + `")`
			}
		}
		if rhsTok != "" && isPathCommand(rhsTok) {
			return indent + firstToken + " = " + `__shell__("` + shellEscape(rhs) + `")`
		}
		if rhsTok != "" && isHyphenatedCommand(rhsTok) {
			return indent + firstToken + " = " + `__shell__("` + shellEscape(rhs) + `")`
		}
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
	// But only if the first token could be a variable (known func/builtin/var or we
	// can't tell), not an unknown command like `ls -la`
	if len(restTrimmed) > 0 && isOperatorStart(restTrimmed[0]) {
		if rugoBuiltins[firstToken] || userFuncs[firstToken] || knownVars[firstToken] {
			return line
		}
		// Unknown ident followed by operator — it's a shell command
		// e.g. `ls -la`, `uname -a`
		return indent + `__shell__("` + shellEscape(trimmed) + `")`
	}

	// Empty rest — bare ident. If it's a known variable, leave it alone (expression).
	// If it's a known function/builtin, it's a no-arg call. Otherwise it's a shell command.
	if restTrimmed == "" {
		if knownVars[firstToken] {
			return line
		}
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

// closestKeywordOrBuiltin returns the closest keyword or builtin to s
// if within edit distance ≤ 2, or "" if none. For short words (≤ 5 chars),
// the first character must match to avoid false positives (e.g. "date" → "rats").
// For very short words (≤ 4 chars), only distance 1 is allowed to prevent
// false positives like "ping" → "print".
func closestKeywordOrBuiltin(s string) string {
	if len(s) < 3 {
		return ""
	}
	best := ""
	bestDist := 3
	check := func(kw string) {
		if len(kw) < 3 {
			return
		}
		// For short words, require first character match to reduce false positives.
		if (len(s) <= 5 || len(kw) <= 5) && s[0] != kw[0] {
			return
		}
		maxDist := 2
		if min(len(s), len(kw)) <= 4 {
			maxDist = 1
		}
		d := util.Levenshtein(s, kw)
		if d > 0 && d <= maxDist && d < bestDist {
			bestDist = d
			best = kw
		}
	}
	for kw := range RugoKeywords {
		check(kw)
	}
	for kw := range rugoBuiltins {
		check(kw)
	}
	return best
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
	sc := NewStringTracker(line)
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InString() {
			continue
		}
		if ch == ' ' && sc.Pos()+3 <= len(line) && line[sc.Pos()+1:sc.Pos()+3] == "or" {
			after := sc.Pos() + 3
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

// expandHashColonSyntax rewrites the colon shorthand for hash keys:
//
// expandDefParens normalizes "def name" (without parentheses) to "def name()"
// so the parser always sees a consistent parameter list. Lines that already
// have parentheses (e.g., "def name(x, y)") are left unchanged.
// Also handles struct method syntax: "def Struct.method" → "def Struct.method()".
func expandDefParens(src string) string {
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "def ") {
			continue
		}
		rest := trimmed[4:]
		// Already has parens — leave it alone
		if strings.Contains(rest, "(") {
			continue
		}
		// Find the function name (may include "Struct.method" dot syntax)
		name := strings.TrimSpace(rest)
		if name == "" || !isIdent(strings.Split(name, ".")[0]) {
			continue
		}
		// Replace "def name" with "def name()" preserving indentation
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		lines[i] = indent + "def " + name + "()"
	}
	return strings.Join(lines, "\n")
}

//	{foo: "bar"}  →  {"foo" => "bar"}
//
// Only bare identifiers followed by ": " are rewritten. String contents
// are left untouched. The arrow syntax {expr => val} is unaffected.
func ExpandHashColonSyntax(src string) (string, error) {
	var sb strings.Builder
	sb.Grow(len(src))

	inDouble := false
	inSingle := false
	inBacktick := false
	escaped := false
	braceDepth := 0
	line := 1

	i := 0
	for i < len(src) {
		ch := src[i]

		if ch == '\n' {
			line++
		}

		if escaped {
			sb.WriteByte(ch)
			escaped = false
			i++
			continue
		}

		if ch == '\\' && (inDouble || inSingle) {
			sb.WriteByte(ch)
			escaped = true
			i++
			continue
		}

		if ch == '"' && !inSingle && !inBacktick {
			inDouble = !inDouble
			sb.WriteByte(ch)
			i++
			continue
		}

		if ch == '\'' && !inDouble && !inBacktick {
			inSingle = !inSingle
			sb.WriteByte(ch)
			i++
			continue
		}

		if ch == '`' && !inDouble && !inSingle {
			inBacktick = !inBacktick
			sb.WriteByte(ch)
			i++
			continue
		}

		if !inDouble && !inSingle && !inBacktick {
			if ch == '{' {
				braceDepth++
			} else if ch == '}' && braceDepth > 0 {
				braceDepth--
			}
		}

		// Outside strings and backticks, look for ident: pattern
		if !inDouble && !inSingle && !inBacktick && (unicode.IsLetter(rune(ch)) || ch == '_') {
			start := i
			for i < len(src) && (unicode.IsLetter(rune(src[i])) || unicode.IsDigit(rune(src[i])) || src[i] == '_') {
				i++
			}
			ident := src[start:i]

			// Check for ident followed by ":" then whitespace
			if i < len(src) && src[i] == ':' && i+1 < len(src) && (src[i+1] == ' ' || src[i+1] == '\t') {
				// Skip colon rewriting on sandbox lines — the colon is part of
				// the sandbox permission syntax (e.g. sandbox ro: "/etc").
				lineStart := strings.LastIndex(src[:start], "\n") + 1
				linePrefix := strings.TrimSpace(src[lineStart:start])
				if linePrefix == "sandbox" || strings.HasPrefix(linePrefix, "sandbox ") {
					sb.WriteString(ident)
					continue
				}
				sb.WriteByte('"')
				sb.WriteString(ident)
				sb.WriteString(`" =>`)
				i++ // skip the ':'
				continue
			}

			sb.WriteString(ident)
			continue
		}

		// Inside hash literal, reject non-identifier colon keys (e.g. {1: "one"})
		if !inDouble && !inSingle && !inBacktick && braceDepth > 0 && unicode.IsDigit(rune(ch)) {
			start := i
			for i < len(src) && unicode.IsDigit(rune(src[i])) {
				i++
			}
			digits := src[start:i]
			if i < len(src) && src[i] == ':' && i+1 < len(src) && (src[i+1] == ' ' || src[i+1] == '\t') {
				return "", fmt.Errorf("line %d: colon syntax only supports identifier keys — use arrow syntax for integer keys: {%s => ...}", line, digits)
			}
			sb.WriteString(digits)
			continue
		}

		sb.WriteByte(ch)
		i++
	}

	return sb.String(), nil
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

// isDottedIdent checks for "ident.ident" format (e.g. "cli.name", "http.get").
func isDottedIdent(s string) bool {
	parts := strings.SplitN(s, ".", 2)
	return len(parts) == 2 && isIdent(parts[0]) && isIdent(parts[1])
}

// isAssignTarget returns true if s is a valid assignment target: a plain
// identifier (e.g. "arr"), or an identifier followed by index/field access
// (e.g. "arr[0]", "h["key"]"). Literals like "42" return false.
func isAssignTarget(s string) bool {
	if isIdent(s) {
		return true
	}
	// Check for identifier followed by index: ident[...]
	if idx := strings.IndexByte(s, '['); idx > 0 {
		return isIdent(s[:idx])
	}
	return false
}

// protectDottedIdent wraps bare dotted idents in parens to prevent the
// preprocessor from treating them as paren-free module calls (e.g. h.x → h.x()).
func protectDottedIdent(expr string) string {
	if isDottedIdent(expr) {
		return "(" + expr + ")"
	}
	return expr
}

// isHyphenatedCommand checks for hyphenated tokens like "docker-compose", "apt-get".
// These start with a letter and contain only ident chars plus hyphens, with at least one hyphen.
// Since hyphens are invalid in Rugo identifiers, these are always shell commands.
func isHyphenatedCommand(s string) bool {
	if len(s) == 0 {
		return false
	}
	ch := rune(s[0])
	if !unicode.IsLetter(ch) && ch != '_' {
		return false
	}
	hasHyphen := false
	for _, c := range s[1:] {
		if c == '-' {
			hasHyphen = true
			continue
		}
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
			return false
		}
	}
	return hasHyphen
}

// isPathCommand checks for tokens that look like filesystem paths:
// `./script`, `../foo`, `/usr/bin/ls`. These are always shell commands.
func isPathCommand(s string) bool {
	return strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "/")
}

func isOperatorStart(ch byte) bool {
	switch ch {
	case '+', '-', '*', '/', '%', '<', '>', '!', '&', '|', '=':
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

// shellEscapePreservingInterpolation escapes shell metacharacters in s
// but leaves #{...} interpolation expressions untouched so the codegen
// string-interpolation pass can process them.
func shellEscapePreservingInterpolation(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '#' && s[i+1] == '{' {
			// Find matching closing brace, respecting nesting.
			depth := 0
			j := i + 1
			for j < len(s) {
				if s[j] == '{' {
					depth++
				} else if s[j] == '}' {
					depth--
					if depth == 0 {
						break
					}
				}
				j++
			}
			if j >= len(s) {
				// Unclosed #{, copy remainder as-is; the parser
				// will report the error downstream.
				sb.WriteString(s[i:])
				break
			}
			// Copy the entire #{...} expression verbatim.
			sb.WriteString(s[i : j+1])
			i = j
			continue
		}
		switch s[i] {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		default:
			sb.WriteByte(s[i])
		}
	}
	return sb.String()
}

// scanFuncDefs does a quick scan to find all `def name(` patterns
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

// expandStructDefs rewrites struct definitions and method definitions.
//
// struct Dog
//
//	name
//	breed
//
// end
//
// becomes:
//
//	def Dog(name, breed)
//	  return {"__type__" => "Dog", "name" => name, "breed" => breed}
//	end
//	def new(name, breed)
//	  return Dog(name, breed)
//	end
//
// And:
//
//	def Dog.bark()
//
// becomes:
//
//	def bark(self)
func ExpandStructDefs(src string) (string, []int, []StructInfo) {
	lines := strings.Split(src, "\n")
	var result []string
	var lineMap []int
	var structs []StructInfo
	structNames := make(map[string]bool)

	// First pass: collect struct names
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "struct ") {
			name := strings.TrimSpace(trimmed[7:])
			if isIdent(name) {
				structNames[name] = true
			}
		}
	}
	singleStruct := len(structNames) == 1

	// Second pass: expand structs and methods
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		origLine := i + 1

		// Expand struct block
		if strings.HasPrefix(trimmed, "struct ") {
			name := strings.TrimSpace(trimmed[7:])
			if !isIdent(name) {
				result = append(result, lines[i])
				lineMap = append(lineMap, origLine)
				i++
				continue
			}

			// Collect field names until "end"
			var fields []string
			i++
			for i < len(lines) {
				ft := strings.TrimSpace(lines[i])
				if ft == "end" {
					i++
					break
				}
				if isIdent(ft) {
					fields = append(fields, ft)
				}
				i++
			}

			structs = append(structs, StructInfo{Name: name, Fields: fields, Line: origLine})

			// Generate constructor: def Name(field1, field2)
			params := strings.Join(fields, ", ")
			var pairs []string
			for _, f := range fields {
				pairs = append(pairs, fmt.Sprintf(`"%s" => %s`, f, f))
			}
			hashBody := `{"__type__" => "` + name + `"`
			if len(pairs) > 0 {
				hashBody += ", " + strings.Join(pairs, ", ")
			}
			hashBody += "}"

			result = append(result, fmt.Sprintf("def %s(%s)", name, params))
			lineMap = append(lineMap, origLine)
			result = append(result, fmt.Sprintf("  return %s", hashBody))
			lineMap = append(lineMap, origLine)
			result = append(result, "end")
			lineMap = append(lineMap, origLine)

			// Generate new() alias only when the file has a single struct
			// to avoid redeclaration when multiple structs share the file.
			if singleStruct {
				result = append(result, fmt.Sprintf("def new(%s)", params))
				lineMap = append(lineMap, origLine)
				result = append(result, fmt.Sprintf("  return %s", hashBody))
				lineMap = append(lineMap, origLine)
				result = append(result, "end")
				lineMap = append(lineMap, origLine)
			}
			continue
		}

		// Expand method definitions: def Name.method(params) → def method(self, params)
		if strings.HasPrefix(trimmed, "def ") {
			rest := strings.TrimSpace(trimmed[4:])
			if dotIdx := strings.Index(rest, "."); dotIdx > 0 {
				typeName := rest[:dotIdx]
				if structNames[typeName] {
					afterDot := rest[dotIdx+1:]
					// Find the opening paren
					parenIdx := strings.Index(afterDot, "(")
					if parenIdx >= 0 {
						methodName := afterDot[:parenIdx]
						paramsStr := afterDot[parenIdx+1:]
						// Remove closing paren if present
						if idx := strings.Index(paramsStr, ")"); idx >= 0 {
							paramsStr = paramsStr[:idx]
						}
						paramsStr = strings.TrimSpace(paramsStr)
						if paramsStr != "" {
							paramsStr = "self, " + paramsStr
						} else {
							paramsStr = "self"
						}
						result = append(result, fmt.Sprintf("def %s(%s)", methodName, paramsStr))
						lineMap = append(lineMap, origLine)
						i++
						continue
					}
				}
			}
		}

		result = append(result, lines[i])
		lineMap = append(lineMap, origLine)
		i++
	}

	return strings.Join(result, "\n"), lineMap, structs
}

// processInterpolation converts "Hello #{expr}" to format string + args.
// Returns the format string and a list of expression strings.
func ProcessInterpolation(s string) (format string, exprs []string, err error) {
	var buf strings.Builder
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
			if depth > 0 {
				return "", nil, fmt.Errorf("unterminated string interpolation — missing closing '}'")
			}
			expr := s[i+2 : j-1]
			exprs = append(exprs, expr)
			buf.WriteString("%v")
			i = j
		} else {
			buf.WriteByte(s[i])
			i++
		}
	}
	return buf.String(), exprs, nil
}

// hasInterpolation checks if a string contains #{} interpolation.
func HasInterpolation(s string) bool {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '#' && s[i+1] == '{' {
			return true
		}
	}
	return false
}

// heredocOpener describes a parsed heredoc opening token (e.g. <<~'DELIM').
type heredocOpener struct {
	delimiter string // e.g. "HTML", "SQL"
	squiggly  bool   // true for <<~ (strip common indentation)
	raw       bool   // true for <<'DELIM' (no interpolation)
}

// parseHeredocOpener tries to parse a heredoc token starting at position pos in line.
// It looks for the pattern <<[~]['"]DELIM['"] where DELIM is [A-Z_][A-Z0-9_]*.
// Returns the parsed opener, the end position (one past the token), and whether
// a valid opener was found.
func parseHeredocOpener(line string, pos int) (heredocOpener, int, bool) {
	i := pos
	if i+2 > len(line) || line[i] != '<' || line[i+1] != '<' {
		return heredocOpener{}, 0, false
	}
	i += 2

	var h heredocOpener

	// Optional squiggly ~
	if i < len(line) && line[i] == '~' {
		h.squiggly = true
		i++
	}

	// Optional single-quote for raw
	quoted := false
	if i < len(line) && line[i] == '\'' {
		h.raw = true
		quoted = true
		i++
	}

	// Delimiter: [A-Z_][A-Z0-9_]*
	start := i
	if i >= len(line) || !(line[i] >= 'A' && line[i] <= 'Z' || line[i] == '_') {
		return heredocOpener{}, 0, false
	}
	for i < len(line) && (line[i] >= 'A' && line[i] <= 'Z' || line[i] >= '0' && line[i] <= '9' || line[i] == '_') {
		i++
	}
	h.delimiter = line[start:i]

	// Closing single-quote for raw
	if quoted {
		if i >= len(line) || line[i] != '\'' {
			return heredocOpener{}, 0, false
		}
		i++
	}

	return h, i, true
}

// findHeredocOpener scans a line for a heredoc token. It matches <<DELIM
// that appears after '=' (assignment context) or after 'return' (return context)
// to avoid ambiguity.
// Returns the opener, the byte offset where the token starts, and whether found.
func findHeredocOpener(line string) (heredocOpener, int, bool) {
	// Check for 'return <<...' context first.
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "return ") || strings.HasPrefix(trimmed, "return\t") {
		// Find "return" in the original line, skip past it + whitespace.
		retIdx := strings.Index(line, "return")
		j := retIdx + len("return")
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		h, end, ok := parseHeredocOpener(line, j)
		if ok {
			rest := strings.TrimSpace(line[end:])
			if rest == "" {
				return h, j, true
			}
		}
	}

	// Look for '=' followed by optional whitespace then <<
	for i := 0; i < len(line); i++ {
		if line[i] == '=' {
			// Skip == and !=
			if i+1 < len(line) && line[i+1] == '=' {
				i++
				continue
			}
			if i > 0 && (line[i-1] == '!' || line[i-1] == '<' || line[i-1] == '>') {
				continue
			}
			// Found assignment '=', skip whitespace after it
			j := i + 1
			for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
				j++
			}
			h, end, ok := parseHeredocOpener(line, j)
			if !ok {
				continue
			}
			// Ensure nothing meaningful follows the opener on this line
			rest := strings.TrimSpace(line[end:])
			if rest != "" {
				continue
			}
			return h, j, true
		}
	}
	return heredocOpener{}, 0, false
}

// stripCommonIndent removes the common leading whitespace from lines,
// ignoring blank lines when computing the minimum indent.
func stripCommonIndent(lines []string) []string {
	minIndent := math.MaxInt
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		indent := 0
		for _, ch := range l {
			if ch == ' ' {
				indent++
			} else if ch == '\t' {
				indent += 4 // treat tab as 4 spaces for indent calculation
			} else {
				break
			}
		}
		if indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent == 0 || minIndent == math.MaxInt {
		return lines
	}

	result := make([]string, len(lines))
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			result[i] = ""
			continue
		}
		// Strip minIndent characters (counting tabs as 4)
		stripped := 0
		j := 0
		for j < len(l) && stripped < minIndent {
			if l[j] == '\t' {
				stripped += 4
			} else {
				stripped++
			}
			j++
		}
		result[i] = l[j:]
	}
	return result
}

// escapeForDoubleQuote escapes a string so it can be embedded inside a
// double-quoted Rugo string literal. Backslashes and double-quotes are escaped.
func escapeForDoubleQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// escapeForSingleQuote escapes a string so it can be embedded inside a
// single-quoted Rugo raw string literal. Backslashes and single-quotes are escaped.
func escapeForSingleQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

// buildHeredocReplacement converts collected heredoc body lines into a
// single-line string expression that the rest of the pipeline can parse.
func buildHeredocReplacement(h heredocOpener, bodyLines []string) string {
	lines := bodyLines
	if h.squiggly {
		lines = stripCommonIndent(lines)
	}

	if h.raw {
		// Raw: concatenate single-quoted segments with "\n" between them.
		// ('line1' + "\n" + 'line2')
		if len(lines) == 0 {
			return "''"
		}
		var parts []string
		for i, l := range lines {
			parts = append(parts, "'"+escapeForSingleQuote(l)+"'")
			if i < len(lines)-1 {
				parts = append(parts, `"\n"`)
			}
		}
		return "(" + strings.Join(parts, " + ") + ")"
	}

	// Interpolating: produce a double-quoted string with \n between lines.
	// "line1\nline2"
	var sb strings.Builder
	sb.WriteByte('"')
	for i, l := range lines {
		sb.WriteString(escapeForDoubleQuote(l))
		if i < len(lines)-1 {
			sb.WriteString(`\n`)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// expandHeredocs replaces heredoc syntax with single-line string expressions.
// Must run before stripComments since heredoc bodies may contain # characters.
//
// Supported forms (DELIM is [A-Z_][A-Z0-9_]*):
//
//	x = <<DELIM          — interpolating heredoc (assignment context)
//	x = <<~DELIM         — interpolating, strip common indent
//	x = <<'DELIM'        — raw heredoc (no interpolation)
//	x = <<~'DELIM'       — raw, strip common indent
//	return <<DELIM       — heredoc in return context
//	return <<~'DELIM'    — raw heredoc in return context
//
// The closing delimiter may be indented; leading whitespace is ignored when
// matching. Body lines between the opener and closer are collected verbatim.
func ExpandHeredocs(src string) (string, []int, error) {
	lines := strings.Split(src, "\n")
	var result []string
	var lineMap []int

	i := 0
	for i < len(lines) {
		h, tokenStart, ok := findHeredocOpener(lines[i])
		if !ok {
			result = append(result, lines[i])
			lineMap = append(lineMap, i+1)
			i++
			continue
		}

		// Replace the <<... token with the expanded string expression later.
		prefix := lines[i][:tokenStart]
		openerLineNum := i + 1

		// Collect body lines until the closing delimiter.
		i++
		var bodyLines []string
		found := false
		for i < len(lines) {
			if strings.TrimSpace(lines[i]) == h.delimiter {
				found = true
				i++
				break
			}
			bodyLines = append(bodyLines, lines[i])
			i++
		}
		if !found {
			return "", nil, fmt.Errorf("%d: unterminated heredoc — missing closing %s (opened at line %d)", openerLineNum, h.delimiter, openerLineNum)
		}

		replacement := buildHeredocReplacement(h, bodyLines)
		result = append(result, prefix+replacement)
		lineMap = append(lineMap, openerLineNum)
	}

	return strings.Join(result, "\n"), lineMap, nil
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
func ExpandTrySugar(src string) (string, []int) {
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
			// Verify it's not == (comparison) and not inside a string literal
			if (eqIdx == 0 || trimmed[eqIdx-1] != '=' && trimmed[eqIdx-1] != '!' && trimmed[eqIdx-1] != '<' && trimmed[eqIdx-1] != '>') && !isInsideString(trimmed, eqIdx) {
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
			// We verify it's truly block form by looking ahead for a body + "end".
			afterOr := strings.TrimSpace(rest[orIdx+2:])
			afterOrTok, afterOrRest := scanFirstToken(afterOr)
			if isIdent(afterOrTok) && !RugoKeywords[afterOrTok] && strings.TrimSpace(afterOrRest) == "" && tryBlockFollows(lines, i+1) {
				// Split the expression onto its own line so preprocessLine can
				// apply shell fallback to bare identifiers inside try.
				expr := strings.TrimSpace(rest[:orIdx])
				expr = protectDottedIdent(expr)
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
			expr = protectDottedIdent(expr)
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
			tryExpr := protectDottedIdent(rest)
			result = append(result, indent+prefix+"try")
			lineMap = append(lineMap, origLine)
			result = append(result, indent+"  "+tryExpr)
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

// tryBlockFollows checks whether lines[start:] begins with at least one
// non-empty body line followed by a line whose trimmed content is "end".
// This disambiguates "try EXPR or ident" (inline fallback) from the
// multi-line block form "try EXPR or err\n  BODY\nend".
func tryBlockFollows(lines []string, start int) bool {
	hasBody := false
	for j := start; j < len(lines); j++ {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "" {
			continue
		}
		if trimmed == "end" {
			return hasBody
		}
		hasBody = true
	}
	return false
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
			if (eqIdx == 0 || trimmed[eqIdx-1] != '=' && trimmed[eqIdx-1] != '!' && trimmed[eqIdx-1] != '<' && trimmed[eqIdx-1] != '>') && !isInsideString(trimmed, eqIdx) {
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

// expandInlineFn expands single-line fn bodies into multi-line form so that
// paren-free calls inside them get preprocessed correctly.
// Example: `arr.each(fn(x) puts x end)` → multi-line with `puts x` on its own line.
// Iterates until no more inline fn bodies remain (handles nested inline fn).
func expandInlineFn(src string, lineMap []int) (string, []int) {
	for {
		lines := strings.Split(src, "\n")
		var result []string
		var newMap []int
		changed := false
		for i, line := range lines {
			origLine := lineMap[i]
			expanded := expandInlineFnLine(line)
			if expanded != line {
				changed = true
				for _, el := range strings.Split(expanded, "\n") {
					result = append(result, el)
					newMap = append(newMap, origLine)
				}
			} else {
				result = append(result, line)
				newMap = append(newMap, origLine)
			}
		}
		src = strings.Join(result, "\n")
		lineMap = newMap
		if !changed {
			break
		}
	}
	return src, lineMap
}

// blockOpenerKeywords are keywords that open a block requiring `end`.
var blockOpenerKeywords = map[string]bool{
	"def": true, "if": true, "while": true, "for": true,
	"try": true, "spawn": true, "parallel": true, "fn": true,
	"rats": true, "bench": true, "struct": true, "do": true,
}

// expandInlineFnLine expands inline fn bodies in a single line.
// It finds `fn(PARAMS) BODY end` where BODY is non-empty and expands to:
//
//	fn(PARAMS)\n  BODY\nend
//
// The function handles strings, nested parens/brackets, and nested block keywords.
// Before expanding, if the line is a paren-free builtin call (e.g. `puts expr`),
// the outer call is wrapped with parens first to prevent the expansion from
// breaking the paren-free rewrite.
func expandInlineFnLine(line string) string {
	// Quick check: does the line contain "fn(" at all?
	if !strings.Contains(line, "fn(") {
		return line
	}

	// Pre-wrap paren-free builtin calls before expanding inline fn.
	// This prevents the expansion from splitting a line like
	// `puts items.map(fn(x) x * 2 end)` into broken fragments.
	line = wrapParenFreeBeforeFnExpand(line)

	// Work through the line, finding inline fn patterns and expanding them.
	// We rebuild the line, replacing each inline fn with its multi-line form.
	var buf strings.Builder
	pos := 0
	changed := false

	for pos < len(line) {
		// Find next "fn(" not inside a string
		fnIdx := findFnOpen(line, pos)
		if fnIdx < 0 {
			buf.WriteString(line[pos:])
			break
		}

		// Write everything before the fn(
		buf.WriteString(line[pos:fnIdx])

		// Find matching ) for the params
		paramStart := fnIdx + 3 // skip "fn("
		paramEnd := findMatchingClose(line, paramStart-1, '(', ')')
		if paramEnd < 0 {
			// No matching ) — leave unchanged
			buf.WriteString(line[fnIdx:])
			break
		}

		params := line[paramStart:paramEnd]
		afterParams := paramEnd + 1 // position after )

		// Check if there's a body before `end` on the same line
		bodyAndRest := line[afterParams:]
		bodyTrimmed := strings.TrimLeft(bodyAndRest, " \t")
		if bodyTrimmed == "" || strings.HasPrefix(bodyTrimmed, "\n") {
			// Already multi-line fn — leave unchanged
			buf.WriteString(line[fnIdx : paramEnd+1])
			pos = paramEnd + 1
			continue
		}

		// Find the matching `end` for this fn, tracking nested blocks
		endIdx := findMatchingEnd(line, afterParams)
		if endIdx < 0 {
			// No matching end found — leave unchanged
			buf.WriteString(line[fnIdx : paramEnd+1])
			pos = paramEnd + 1
			continue
		}

		body := strings.TrimSpace(line[afterParams:endIdx])
		if body == "" {
			// Empty body — leave as-is
			buf.WriteString(line[fnIdx : endIdx+3]) // include "end"
			pos = endIdx + 3
			continue
		}

		// Expand: fn(PARAMS)\n  BODY\nend
		buf.WriteString("fn(" + params + ")\n  " + body + "\nend")
		pos = endIdx + 3 // skip past "end"
		changed = true
	}

	if !changed {
		return line
	}
	return buf.String()
}

// wrapParenFreeBeforeFnExpand wraps paren-free builtin calls with explicit parens
// so that subsequent inline fn expansion doesn't produce broken lines.
// For example: `puts items.map(fn(x) x * 2 end)` → `puts(items.map(fn(x) x * 2 end))`
func wrapParenFreeBeforeFnExpand(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line
	}
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	firstToken, rest := scanFirstToken(trimmed)
	if firstToken == "" || !isIdent(firstToken) {
		return line
	}
	// Only wrap if the first token is a known builtin
	if !rugoBuiltins[firstToken] {
		return line
	}
	restTrimmed := strings.TrimSpace(rest)
	if restTrimmed == "" {
		return line
	}
	// Already has parens: `puts(...)` — leave alone
	if restTrimmed[0] == '(' {
		return line
	}
	// Assignment or operator — leave alone
	if restTrimmed[0] == '=' && (len(restTrimmed) < 2 || restTrimmed[1] != '=') {
		return line
	}
	// Wrap: `puts expr` → `puts(expr)`
	return indent + firstToken + "(" + restTrimmed + ")"
}

// findFnOpen finds the next "fn(" in line starting from pos that is not inside a string.
func findFnOpen(line string, pos int) int {
	inDouble := false
	inSingle := false
	escaped := false
	for i := pos; i < len(line); i++ {
		ch := line[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && (inDouble || inSingle) {
			escaped = true
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if inDouble || inSingle {
			continue
		}
		if i+3 <= len(line) && line[i:i+3] == "fn(" {
			// Make sure "fn" is not part of a larger identifier
			if i > 0 && (isAlphaNum(line[i-1]) || line[i-1] == '_') {
				continue
			}
			return i
		}
	}
	return -1
}

// findMatchingClose finds the position of the matching closing bracket
// starting from an open bracket at position openPos.
// Returns the position of the closing bracket, or -1 if not found.
func findMatchingClose(line string, openPos int, open, close byte) int {
	depth := 0
	inDouble := false
	inSingle := false
	escaped := false
	for i := openPos; i < len(line); i++ {
		ch := line[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && (inDouble || inSingle) {
			escaped = true
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if inDouble || inSingle {
			continue
		}
		if ch == open {
			depth++
		} else if ch == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// findMatchingEnd finds the position of the `end` keyword that matches
// the fn block, starting from startPos (after the fn params close paren).
// Tracks nested block keywords to find the correct matching end.
// Returns the start position of the matching "end", or -1.
func findMatchingEnd(line string, startPos int) int {
	depth := 1 // we're inside the fn block
	inDouble := false
	inSingle := false
	escaped := false
	i := startPos

	for i < len(line) {
		ch := line[i]
		if escaped {
			escaped = false
			i++
			continue
		}
		if ch == '\\' && (inDouble || inSingle) {
			escaped = true
			i++
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			i++
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			i++
			continue
		}
		if inDouble || inSingle {
			i++
			continue
		}

		// Check for word boundaries: extract the current word
		if isAlpha(ch) || ch == '_' {
			wordStart := i
			for i < len(line) && (isAlphaNum(line[i]) || line[i] == '_') {
				i++
			}
			word := line[wordStart:i]

			// Check word boundary: must not be preceded by alphanumeric/_
			preceded := wordStart > 0 && (isAlphaNum(line[wordStart-1]) || line[wordStart-1] == '_')
			if preceded {
				continue
			}

			if word == "end" {
				depth--
				if depth == 0 {
					return wordStart
				}
			} else if blockOpenerKeywords[word] {
				depth++
			}
			continue
		}
		i++
	}
	return -1
}

func isAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isAlphaNum(ch byte) bool {
	return isAlpha(ch) || (ch >= '0' && ch <= '9')
}

// isInsideString reports whether position pos in line falls inside a string literal.
func isInsideString(line string, pos int) bool {
	return IsInsideString(line, pos)
}

// findTopLevelOr finds " or " at the top level (not inside parens, brackets, or strings).
// Returns the index of the start of " or " in s, or -1 if not found.
func findTopLevelOr(s string) int {
	pos := FindTopLevel(s, func(ch byte, pos int, src string) bool {
		if ch != ' ' {
			return false
		}
		return pos+4 <= len(src) && src[pos:pos+4] == " or "
	})
	if pos < 0 {
		return -1
	}
	return pos + 1 // return index of 'o' in "or"
}

// expandDestructuring desugars array destructuring assignments.
//
//	a, b = expr       → __destr__ = expr; a = __destr__[0]; b = __destr__[1]
//	a, b, c = expr    → __destr__ = expr; a = __destr__[0]; b = __destr__[1]; c = __destr__[2]
//
// Only matches lines where the LHS is two or more comma-separated identifiers
// followed by `=`. Does not match inside strings, and skips lines starting with
// keywords (for, def, etc.).
func expandDestructuring(src string) string {
	lines := strings.Split(src, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, line)
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		// Skip keywords that use commas (for k, v in ...; def foo(a, b); etc.)
		first, _ := scanFirstToken(trimmed)
		if RugoKeywords[first] {
			result = append(result, line)
			continue
		}

		// Look for the pattern: ident, ident [, ident]* = expr
		// Find the top-level `=` that is not `==`
		eqIdx := findDestructAssign(trimmed)
		if eqIdx < 0 {
			result = append(result, line)
			continue
		}

		lhs := strings.TrimSpace(trimmed[:eqIdx])
		rhs := strings.TrimSpace(trimmed[eqIdx+1:])

		// LHS must contain at least one comma
		if !strings.Contains(lhs, ",") {
			result = append(result, line)
			continue
		}

		// Split LHS into identifiers and validate each one
		parts := strings.Split(lhs, ",")
		if len(parts) < 2 {
			result = append(result, line)
			continue
		}
		targets := make([]string, 0, len(parts))
		valid := true
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if !isIdent(p) {
				valid = false
				break
			}
			targets = append(targets, p)
		}
		if !valid {
			result = append(result, line)
			continue
		}

		// Emit: __destr__ = expr
		result = append(result, indent+"__destr__ = "+rhs)
		// Emit: target = __destr__[i] for each target
		for i, t := range targets {
			result = append(result, fmt.Sprintf("%s%s = __destr__[%d]", indent, t, i))
		}
	}
	return strings.Join(result, "\n")
}

// findDestructAssign finds a top-level `=` (not `==`, `!=`, `<=`, `>=`, `=>`)
// in a line, skipping content inside strings, parens, and brackets.
// Returns the index of `=` or -1.
func findDestructAssign(s string) int {
	return FindTopLevel(s, func(ch byte, pos int, src string) bool {
		if ch != '=' {
			return false
		}
		if pos+1 < len(src) && (src[pos+1] == '=' || src[pos+1] == '>') {
			return false
		}
		if pos > 0 && (src[pos-1] == '!' || src[pos-1] == '<' || src[pos-1] == '>') {
			return false
		}
		return true
	})
}

// expandCompoundAssign desugars compound assignment operators.
//
//	x += y       → x = x + y
//	arr[0] += y  → arr[0] = arr[0] + y
//
// Handles +=, -=, *=, /=, %=. Respects string boundaries.
func ExpandCompoundAssign(src string) string {
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
	return FindTopLevel(s, func(ch byte, pos int, src string) bool {
		return pos+len(op) <= len(src) && src[pos:pos+len(op)] == op
	})
}

// expandBareAppend desugars bare append statements.
//
//	append(x, val)  → x = append(x, val)
//
// Only rewrites when append( starts the line (after whitespace). Lines like
// "y = append(x, val)" or "puts(append(x, val))" are left alone.
func ExpandBareAppend(src string) string {
	lines := strings.Split(src, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "append(") {
			result = append(result, line)
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		// Extract the first argument from append(firstArg, rest...)
		inner := trimmed[len("append("):]
		firstArg := extractFirstArg(inner)
		if firstArg == "" || !isAssignTarget(firstArg) {
			result = append(result, line)
			continue
		}
		result = append(result, indent+firstArg+" = "+trimmed)
	}
	return strings.Join(result, "\n")
}

// extractFirstArg returns the first argument from a comma-separated argument
// list, respecting balanced parens, brackets, braces, and string boundaries.
// Returns "" if no comma separator is found.
func extractFirstArg(s string) string {
	pos := FindTopLevel(s, func(ch byte, pos int, src string) bool {
		return ch == ','
	})
	if pos < 0 {
		return ""
	}
	return strings.TrimSpace(s[:pos])
}

// expandBackticks converts `cmd` expressions to __capture__("cmd") calls.
// Backticks inside string literals are left untouched.
// Returns an error if an unclosed backtick is found.
func expandBackticks(src string) (string, error) {
	var sb strings.Builder
	inDouble := false
	inSingle := false
	escaped := false
	lineNum := 1
	for i := 0; i < len(src); i++ {
		ch := src[i]
		if ch == '\n' {
			lineNum++
		}
		if escaped {
			sb.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' && (inDouble || inSingle) {
			sb.WriteByte(ch)
			escaped = true
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			sb.WriteByte(ch)
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			sb.WriteByte(ch)
			continue
		}
		if ch == '`' && !inDouble && !inSingle {
			btLine := lineNum
			// Find the closing backtick
			j := i + 1
			for j < len(src) && src[j] != '`' {
				j++
			}
			if j >= len(src) {
				return "", fmt.Errorf("%d: unterminated backtick expression (opened at line %d)", btLine, btLine)
			}
			cmd := src[i+1 : j]
			sb.WriteString(`__capture__("` + shellEscapePreservingInterpolation(cmd) + `")`)
			i = j
			continue
		}
		sb.WriteByte(ch)
	}
	return sb.String(), nil
}

// rugoVoidBuiltins are builtins that return nil. Using them as non-final
// segments in a pipe chain is almost certainly a mistake (the downstream
// segments would receive nil).
var rugoVoidBuiltins = map[string]bool{
	"puts": true, "print": true,
}

// expandPipeLine detects top-level | operators in a line and rewrites them
// into function calls. A | connects the output of the left side to the input
// of the right side:
//   - Shell command on left → captured stdout (like backticks)
//   - Function/expr on left → return value
//   - Function on right → piped value becomes first argument
//   - Shell command on right → piped value fed to stdin
//
// If ALL segments are shell commands, the line is returned unchanged so the
// shell handles native pipes (e.g. `ls | grep foo`).
// Returns an error if a void-returning builtin (puts, print) appears as a
// non-final segment.
func expandPipeLine(line string, funcs map[string]bool) (string, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line, nil
	}
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	// Don't expand pipes on keyword-prefixed lines
	firstTok, _ := scanFirstToken(trimmed)
	if RugoKeywords[firstTok] {
		return line, nil
	}

	// Extract assignment prefix: "x = EXPR" → prefix="x = ", expr="EXPR"
	prefix, expr := extractPipeAssignPrefix(trimmed)

	// Find top-level pipe positions (not ||, not inside strings/parens/brackets)
	pipes := findTopLevelPipes(expr)
	if len(pipes) == 0 {
		return line, nil
	}

	// Split into segments
	segments := splitAtPositions(expr, pipes)

	// If ALL segments are shell commands, return unchanged (shell handles native pipes)
	hasRugo := false
	for _, seg := range segments {
		if isRugoSegment(strings.TrimSpace(seg), funcs) {
			hasRugo = true
			break
		}
	}
	if !hasRugo {
		return line, nil
	}

	// Validate: void-returning builtins (puts, print) in non-final position
	// break the chain since they return nil.
	for i := 0; i < len(segments)-1; i++ {
		seg := strings.TrimSpace(segments[i])
		tok, _ := scanFirstToken(seg)
		if rugoVoidBuiltins[tok] {
			return "", fmt.Errorf("`%s` returns nil — piping it further discards results; move `%s` to the end of the pipe chain", tok, tok)
		}
	}

	// Build the piped expression
	result := buildPipedExpr(segments, funcs)
	return indent + prefix + result, nil
}

// extractPipeAssignPrefix detects simple "ident = EXPR" assignment and returns
// the prefix and expression parts. Returns ("", fullLine) if no assignment found.
func extractPipeAssignPrefix(trimmed string) (string, string) {
	tok, rest := scanFirstToken(trimmed)
	if tok == "" || RugoKeywords[tok] || !isIdent(tok) {
		return "", trimmed
	}
	restTrimmed := strings.TrimSpace(rest)
	if len(restTrimmed) > 0 && restTrimmed[0] == '=' &&
		(len(restTrimmed) < 2 || restTrimmed[1] != '=') {
		expr := strings.TrimSpace(restTrimmed[1:])
		return tok + " = ", expr
	}
	return "", trimmed
}

// findTopLevelPipes finds positions of | characters that are pipe operators
// (not part of ||, not inside strings, parens, or brackets).
func findTopLevelPipes(s string) []int {
	return FindAllTopLevel(s, func(ch byte, pos int, src string) bool {
		if ch != '|' {
			return false
		}
		if pos+1 < len(src) && src[pos+1] == '|' {
			return false
		}
		if pos > 0 && src[pos-1] == '|' {
			return false
		}
		return true
	})
}

// splitAtPositions splits a string at the given positions, excluding the
// character at each position.
func splitAtPositions(s string, positions []int) []string {
	var segments []string
	prev := 0
	for _, pos := range positions {
		segments = append(segments, s[prev:pos])
		prev = pos + 1
	}
	segments = append(segments, s[prev:])
	return segments
}

// isRugoSegment returns true if the segment is a Rugo construct (function call,
// builtin, dotted ident, literal) rather than a shell command.
func isRugoSegment(seg string, funcs map[string]bool) bool {
	if seg == "" {
		return false
	}
	firstTok, _ := scanFirstToken(seg)
	if firstTok == "" {
		return false
	}
	if rugoBuiltins[firstTok] || funcs[firstTok] {
		return true
	}
	// Internal functions produced by earlier preprocessor passes.
	if firstTok == "__capture__" || firstTok == "__shell__" || firstTok == "__pipe_shell__" {
		return true
	}
	if isDottedIdent(firstTok) {
		return true
	}
	// Starts with non-identifier char (string literal, number, paren) → Rugo expr
	if !isIdent(firstTok) && !isHyphenatedCommand(firstTok) {
		return true
	}
	return false
}

// isShellPipeSegment returns true if the segment should be treated as a shell
// command in a pipe chain.
func isShellPipeSegment(seg string, funcs map[string]bool) bool {
	return !isRugoSegment(seg, funcs)
}

// buildPipedExpr builds the final expression from pipe segments.
// Each segment's output becomes the input of the next.
func buildPipedExpr(segments []string, funcs map[string]bool) string {
	first := strings.TrimSpace(segments[0])
	var acc string

	if isShellPipeSegment(first, funcs) {
		acc = `__capture__("` + shellEscape(first) + `")`
	} else {
		acc = segmentToExpr(first, funcs)
	}

	for i := 1; i < len(segments); i++ {
		seg := strings.TrimSpace(segments[i])
		if isShellPipeSegment(seg, funcs) {
			acc = `__pipe_shell__("` + shellEscape(seg) + `", ` + acc + `)`
		} else {
			acc = segmentWithPipedArg(seg, acc, funcs)
		}
	}

	return acc
}

// segmentToExpr converts a pipe segment to a Rugo expression, adding parens
// for paren-free calls.
func segmentToExpr(seg string, funcs map[string]bool) string {
	firstTok, rest := scanFirstToken(seg)
	restTrimmed := strings.TrimSpace(rest)

	if rugoBuiltins[firstTok] || funcs[firstTok] || isDottedIdent(firstTok) {
		if restTrimmed == "" {
			return firstTok + "()"
		}
		if len(restTrimmed) > 0 && restTrimmed[0] == '(' {
			return seg
		}
		return firstTok + "(" + restTrimmed + ")"
	}
	return seg
}

// segmentWithPipedArg wraps a Rugo function/builtin call with the piped value
// prepended as the first argument.
func segmentWithPipedArg(seg string, piped string, funcs map[string]bool) string {
	firstTok, rest := scanFirstToken(seg)
	restTrimmed := strings.TrimSpace(rest)

	if rugoBuiltins[firstTok] || funcs[firstTok] || isDottedIdent(firstTok) {
		if restTrimmed == "" {
			return firstTok + "(" + piped + ")"
		}
		if len(restTrimmed) > 0 && restTrimmed[0] == '(' {
			if restTrimmed == "()" {
				return firstTok + "(" + piped + ")"
			}
			// func(args...) → func(piped, args...)
			return firstTok + "(" + piped + ", " + restTrimmed[1:]
		}
		// Paren-free: func arg1, arg2 → func(piped, arg1, arg2)
		return firstTok + "(" + piped + ", " + restTrimmed + ")"
	}
	// Fallback: treat as function call
	return firstTok + "(" + piped + ")"
}

// RejectSemicolons scans source for user-written semicolons outside strings
// and heredocs, and returns an error if found. Semicolons are reserved for
// internal use by the preprocessor as statement separators.
func RejectSemicolons(src string) error {
	inDouble := false
	inSingle := false
	escaped := false
	line := 1
	lines := strings.Split(src, "\n")
	// Build a set of lines that are inside heredoc bodies so we can skip them.
	inHeredoc := make(map[int]bool)
	var heredocDelim string
	for li, ln := range lines {
		lineNum := li + 1
		if heredocDelim != "" {
			inHeredoc[lineNum] = true
			if strings.TrimSpace(ln) == heredocDelim {
				heredocDelim = ""
			}
			continue
		}
		// Check if this line opens a heredoc.
		if _, _, ok := findHeredocOpener(ln); ok {
			h, _, _ := findHeredocOpener(ln)
			heredocDelim = h.delimiter
		}
	}
	for i := 0; i < len(src); i++ {
		ch := src[i]
		if ch == '\n' {
			line++
		}
		if inHeredoc[line] {
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && (inDouble || inSingle) {
			escaped = true
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == ';' && !inDouble && !inSingle {
			return fmt.Errorf("line %d: semicolons are not supported in Rugo", line)
		}
	}
	return nil
}

// RejectTrailingCommas scans source for trailing commas before ']' or '}'
// (outside string literals) and returns an error if found.
func RejectTrailingCommas(src string) error {
	inString := false
	var strChar byte
	line := 1
	commaLine := 0
	seenComma := false
	for i := 0; i < len(src); i++ {
		ch := src[i]
		if ch == '\n' {
			line++
		}
		if inString {
			if ch == '\\' {
				i++ // skip escaped character
			} else if ch == strChar {
				inString = false
			}
			continue
		}
		switch ch {
		case '"', '\'':
			inString = true
			strChar = ch
			seenComma = false
		case '#':
			// skip rest of line (comment)
			for i+1 < len(src) && src[i+1] != '\n' {
				i++
			}
		case ',':
			seenComma = true
			commaLine = line
		case ']', '}':
			if seenComma {
				return fmt.Errorf("line %d: trailing commas are not allowed", commaLine)
			}
		case ' ', '\t', '\n', '\r':
			// whitespace doesn't reset seenComma
		default:
			seenComma = false
		}
	}
	return nil
}

// InsertArraySeparators inserts ';' before lines that start with '[' (after
// optional whitespace) to disambiguate array literals from index suffix
// operations across line boundaries. It tracks bracket depth so that lines
// inside a multi-line array (e.g. nested array elements) are not separated.
// insertSandboxSeparators appends a ';' after lines that start with `sandbox`
// so the parser can distinguish the end of the sandbox statement from the next
// statement on the following line.
func insertSandboxSeparators(src string) string {
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "sandbox" || strings.HasPrefix(trimmed, "sandbox ") {
			lines[i] = lines[i] + " ;"
		}
	}
	return strings.Join(lines, "\n")
}

func InsertArraySeparators(src string) string {
	lines := strings.Split(src, "\n")
	bracketDepth := 0
	for i := 0; i < len(lines); i++ {
		if i > 0 && bracketDepth == 0 {
			trimmed := strings.TrimSpace(lines[i])
			if len(trimmed) > 0 && trimmed[0] == '[' {
				indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				lines[i] = indent + ";" + strings.TrimLeft(lines[i], " \t")
			}
		}
		bracketDepth += countBracketDelta(lines[i])
	}
	return strings.Join(lines, "\n")
}

// InsertNegativeLiteralSeparators inserts ';' before lines that start with '-'
// followed by a digit, so the parser treats them as negative literals (unary
// negation) rather than binary subtraction from the previous expression.
func InsertNegativeLiteralSeparators(src string) string {
	lines := strings.Split(src, "\n")
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if len(trimmed) >= 2 && trimmed[0] == '-' && trimmed[1] >= '0' && trimmed[1] <= '9' {
			indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
			lines[i] = indent + ";" + strings.TrimLeft(lines[i], " \t")
		}
	}
	return strings.Join(lines, "\n")
}

// countBracketDelta returns the net change in bracket depth for a line,
// skipping brackets inside string literals.
func countBracketDelta(line string) int {
	delta := 0
	inString := false
	var strChar byte
	for j := 0; j < len(line); j++ {
		ch := line[j]
		if inString {
			if ch == '\\' {
				j++ // skip escaped character
			} else if ch == strChar {
				inString = false
			}
			continue
		}
		switch ch {
		case '"', '\'':
			inString = true
			strChar = ch
		case '#':
			return delta // rest of line is a comment
		case '[':
			delta++
		case ']':
			delta--
		}
	}
	return delta
}

// expandDoEnd rewrites `do...end` trailing blocks into `fn()...end` syntax.
//
// Patterns:
//
//	vbox do              →  vbox(fn()
//	  label "Hello"           label "Hello"
//	end                     end)
//
//	button("Click") do  →  button("Click", fn()
//	  puts "clicked"          puts "clicked"
//	end                     end)
//
//	app "Title", 400 do →  app("Title", 400, fn()
//	  ...                     ...
//	end                     end)
//
// The pass runs before try sugar, spawn sugar, and inline fn expansion.
func expandDoEnd(src string) (string, error) {
	// Iterate until no more do...end blocks remain (handles nesting).
	for {
		lines := strings.Split(src, "\n")
		var result []string
		changed := false
		i := 0
		for i < len(lines) {
			line := lines[i]
			prefix, isDo := extractDoPrefix(line)
			if !isDo {
				result = append(result, line)
				i++
				continue
			}

			// Find matching end
			endIdx := findDoMatchingEnd(lines, i+1)
			if endIdx < 0 {
				return "", fmt.Errorf("unterminated `do` block (opened at line %d)", i+1)
			}

			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			endIndent := lines[endIdx][:len(lines[endIdx])-len(strings.TrimLeft(lines[endIdx], " \t"))]

			// Rewrite opening line
			result = append(result, rewriteDoPrefix(prefix, indent))

			// Copy body lines unchanged
			for j := i + 1; j < endIdx; j++ {
				result = append(result, lines[j])
			}

			// Rewrite end line: `end` → `end)`
			result = append(result, endIndent+"end)")

			changed = true
			i = endIdx + 1
		}
		src = strings.Join(result, "\n")
		if !changed {
			break
		}
	}
	return src, nil
}

// extractDoPrefix checks if a line ends with ` do` (not inside a string)
// and returns the prefix (everything before ` do`) and true, or ("", false).
func extractDoPrefix(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 4 { // minimum: "x do"
		return "", false
	}
	if !strings.HasSuffix(trimmed, " do") && !strings.HasSuffix(trimmed, "\tdo") {
		return "", false
	}
	// Position of 'd' in the trailing "do"
	doPos := len(trimmed) - 2
	// Ensure "do" is not inside a string
	if isInsideString(trimmed, doPos) {
		return "", false
	}
	prefix := strings.TrimSpace(trimmed[:doPos])
	if prefix == "" {
		return "", false // bare `do` with no function call
	}
	return prefix, true
}

// findDoMatchingEnd finds the line index of the `end` that matches a `do` block,
// starting search from startLine. Tracks nested blocks including fn() lambdas
// that may appear mid-line (e.g., "x = fn() ... end").
func findDoMatchingEnd(lines []string, startLine int) int {
	depth := 1
	for i := startLine; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}

		// Check for do...end on this line (nested do block opener)
		if _, isDo := extractDoPrefix(lines[i]); isDo {
			depth++
			continue
		}

		first, _ := scanFirstToken(trimmed)

		// Count fn( occurrences on this line (opens blocks mid-line)
		depth += countFnOpens(trimmed)

		// Check for other block openers at line start
		switch first {
		case "def", "if", "while", "for", "try", "spawn", "parallel",
			"rats", "bench", "struct":
			depth++
		}

		// Count all `end` keywords on this line (handles "end)" from prior do expansion)
		depth -= countEnds(trimmed)
		if depth <= 0 {
			return i
		}
	}
	return -1
}

// countFnOpens counts how many `fn(` block openers appear on a line,
// skipping content inside strings.
func countFnOpens(line string) int {
	count := 0
	inDouble := false
	inSingle := false
	escaped := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && (inDouble || inSingle) {
			escaped = true
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if inDouble || inSingle {
			continue
		}
		if i+3 <= len(line) && line[i:i+3] == "fn(" {
			if i > 0 && (isAlphaNum(line[i-1]) || line[i-1] == '_') {
				continue
			}
			count++
		}
	}
	return count
}

// countEnds counts how many standalone `end` keywords appear on a line,
// at word boundaries and not inside strings. Handles `end` and `end)`.
func countEnds(line string) int {
	count := 0
	inDouble := false
	inSingle := false
	escaped := false
	i := 0
	for i < len(line) {
		ch := line[i]
		if escaped {
			escaped = false
			i++
			continue
		}
		if ch == '\\' && (inDouble || inSingle) {
			escaped = true
			i++
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			i++
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			i++
			continue
		}
		if inDouble || inSingle {
			i++
			continue
		}
		if ch == 'e' && i+3 <= len(line) && line[i:i+3] == "end" {
			// Check word boundaries
			before := i == 0 || !(isAlphaNum(line[i-1]) || line[i-1] == '_')
			after := i+3 >= len(line) || !(isAlphaNum(line[i+3]) || line[i+3] == '_')
			if before && after {
				count++
				i += 3
				continue
			}
		}
		i++
	}
	return count
}

// rewriteDoPrefix rewrites the call prefix to inject `fn()` as the last argument.
//
//	"vbox"                    → "vbox(fn()"
//	"button(\"Click\")"      → "button(\"Click\", fn()"
//	"app \"Title\", 400"     → "app(\"Title\", 400, fn()"
func rewriteDoPrefix(prefix, indent string) string {
	trimmed := strings.TrimSpace(prefix)

	// Handle assignment: "x = expr"
	assignPart := ""
	callPart := trimmed
	if eqIdx := findDoAssignment(trimmed); eqIdx >= 0 {
		assignPart = strings.TrimSpace(trimmed[:eqIdx+1]) + " "
		callPart = strings.TrimSpace(trimmed[eqIdx+1:])
	}

	// If call ends with ")", insert fn() before the closing paren
	if strings.HasSuffix(callPart, ")") {
		// Find the matching opening paren for the last ")"
		lastParen := len(callPart) - 1
		return indent + assignPart + callPart[:lastParen] + ", fn()"
	}

	// No parens: bare ident or paren-free call
	firstToken, rest := scanFirstToken(callPart)
	restTrimmed := strings.TrimSpace(rest)
	if restTrimmed == "" {
		// Bare ident: `vbox` → `vbox(fn()`
		return indent + assignPart + firstToken + "(fn()"
	}
	// Paren-free: `app "Title", 400` → `app("Title", 400, fn()`
	return indent + assignPart + firstToken + "(" + restTrimmed + ", fn()"
}

// findDoAssignment finds a top-level `=` (not `==`, `!=`, `<=`, `>=`, `=>`)
// suitable for splitting assignment from a do-block call prefix.
func findDoAssignment(s string) int {
	return FindTopLevel(s, func(ch byte, pos int, src string) bool {
		if ch != '=' {
			return false
		}
		if pos+1 < len(src) && (src[pos+1] == '=' || src[pos+1] == '>') {
			return false
		}
		if pos > 0 && (src[pos-1] == '!' || src[pos-1] == '<' || src[pos-1] == '>') {
			return false
		}
		return true
	})
}

// StructInfo holds metadata about a struct definition extracted during
// preprocessing. Structs are expanded into constructor functions before
// parsing, so they don't appear in the AST as nodes.
type StructInfo struct {
	Name   string   // struct name (e.g. "Dog")
	Fields []string // field names
	Line   int      // 1-based line number of the struct keyword in original source
}

// blockStartKeywords are tokens that, when they start a line, indicate
// a block-level statement. Lines starting with these are never postfix-if.
var blockStartKeywords = map[string]bool{
	"if": true, "elsif": true, "else": true, "end": true,
	"while": true, "for": true, "def": true, "return": true,
	"require": true, "import": true, "use": true,
	"rats": true, "try": true, "spawn": true, "parallel": true,
	"bench": true, "fn": true, "struct": true, "sandbox": true,
	"setup": true, "teardown": true, "setup_file": true, "teardown_file": true,
}

// expandPostfixIf rewrites "STMT if COND" → "if COND\nSTMT\nend".
// Only matches when " if " appears at the top level (outside strings and
// brackets) and the line does not start with a block keyword.
func expandPostfixIf(src string) string {
	lines := strings.Split(src, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		firstTok, _ := scanFirstToken(trimmed)
		if blockStartKeywords[firstTok] || trimmed == "" {
			result = append(result, line)
			continue
		}

		// Find the last top-level " if " on the line.
		ifIdx := findLastTopLevelIf(trimmed)
		if ifIdx < 0 {
			result = append(result, line)
			continue
		}

		stmt := strings.TrimSpace(trimmed[:ifIdx])
		cond := strings.TrimSpace(trimmed[ifIdx+4:])
		if stmt == "" || cond == "" {
			result = append(result, line)
			continue
		}

		result = append(result, indent+"if "+cond)
		result = append(result, indent+"  "+stmt)
		result = append(result, indent+"end")
	}
	return strings.Join(result, "\n")
}

// findLastTopLevelIf returns the byte offset of the last " if " token
// that appears at the top level (outside strings, backticks, and brackets).
// Returns -1 if none found.
func findLastTopLevelIf(line string) int {
	positions := FindAllTopLevel(line, func(ch byte, pos int, src string) bool {
		if ch != ' ' {
			return false
		}
		// Check for " if " starting at this position
		if pos+4 > len(src) {
			return false
		}
		return src[pos:pos+4] == " if " || (src[pos:pos+3] == " if" && pos+3 == len(src))
	})
	if len(positions) == 0 {
		return -1
	}
	return positions[len(positions)-1]
}
