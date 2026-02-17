package preprocess

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringTracker_BasicIteration(t *testing.T) {
	sc := NewStringTracker("abc")
	ch, ok := sc.Next()
	require.True(t, ok)
	assert.Equal(t, byte('a'), ch)
	assert.Equal(t, 0, sc.Pos())

	ch, ok = sc.Next()
	require.True(t, ok)
	assert.Equal(t, byte('b'), ch)

	ch, ok = sc.Next()
	require.True(t, ok)
	assert.Equal(t, byte('c'), ch)

	_, ok = sc.Next()
	assert.False(t, ok)
}

func TestStringTracker_LineTracking(t *testing.T) {
	sc := NewStringTracker("a\nb\nc")
	sc.Next() // a
	assert.Equal(t, 1, sc.Line())
	sc.Next() // \n
	assert.Equal(t, 2, sc.Line())
	sc.Next() // b
	assert.Equal(t, 2, sc.Line())
	sc.Next() // \n
	assert.Equal(t, 3, sc.Line())
	sc.Next() // c
	assert.Equal(t, 3, sc.Line())
}

func TestStringTracker_DoubleQuotedString(t *testing.T) {
	sc := NewStringTracker(`x = "hello" + y`)
	var codeBytes, strBytes []byte
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InString() {
			strBytes = append(strBytes, ch)
		} else {
			codeBytes = append(codeBytes, ch)
		}
	}
	assert.Equal(t, `x =  + y`, string(codeBytes))
	assert.Equal(t, `"hello"`, string(strBytes))
}

func TestStringTracker_SingleQuotedString(t *testing.T) {
	sc := NewStringTracker(`x = 'hello' + y`)
	var strBytes []byte
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InString() {
			strBytes = append(strBytes, ch)
		}
	}
	assert.Equal(t, `'hello'`, string(strBytes))
}

func TestStringTracker_BacktickString(t *testing.T) {
	input := "x = `ls -la` + y"
	sc := NewStringTracker(input)
	var strBytes []byte
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InBacktick() {
			strBytes = append(strBytes, ch)
		}
	}
	assert.Equal(t, "`ls -la`", string(strBytes))
}

func TestStringTracker_EscapedQuotes(t *testing.T) {
	// "he\"llo" — the escaped quote should NOT end the string
	sc := NewStringTracker(`"he\"llo" + x`)
	var strBytes []byte
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InString() {
			strBytes = append(strBytes, ch)
		}
	}
	assert.Equal(t, `"he\"llo"`, string(strBytes))
}

func TestStringTracker_EscapedSingleQuotes(t *testing.T) {
	sc := NewStringTracker(`'he\'llo' + x`)
	var strBytes []byte
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InString() {
			strBytes = append(strBytes, ch)
		}
	}
	assert.Equal(t, `'he\'llo'`, string(strBytes))
}

func TestStringTracker_NestedQuotes(t *testing.T) {
	// Double quotes inside single quotes should not toggle double state
	sc := NewStringTracker(`'"hello"' + x`)
	var strBytes []byte
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InString() {
			strBytes = append(strBytes, ch)
		}
	}
	assert.Equal(t, `'"hello"'`, string(strBytes))
}

func TestStringTracker_BacktickInsideDoubleQuote(t *testing.T) {
	sc := NewStringTracker("\"hello `world`\" + x")
	var strBytes []byte
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InString() {
			strBytes = append(strBytes, ch)
		}
	}
	assert.Equal(t, "\"hello `world`\"", string(strBytes))
}

func TestStringTracker_Peek(t *testing.T) {
	sc := NewStringTracker("ab")
	ch, ok := sc.Peek()
	require.True(t, ok)
	assert.Equal(t, byte('a'), ch)
	assert.Equal(t, -1, sc.Pos()) // Peek doesn't advance

	sc.Next() // a
	ch, ok = sc.Peek()
	require.True(t, ok)
	assert.Equal(t, byte('b'), ch)

	sc.Next() // b
	_, ok = sc.Peek()
	assert.False(t, ok)
}

func TestStringTracker_LookingAt(t *testing.T) {
	sc := NewStringTracker("fn(x) end")
	sc.Next() // f - pos 0
	assert.True(t, sc.LookingAt("fn("))
	assert.False(t, sc.LookingAt("end"))

	sc.Skip(5) // skip n(x)[space] - pos 5
	sc.Next()  // e - pos 6
	assert.True(t, sc.LookingAt("end"))
}

func TestStringTracker_Skip(t *testing.T) {
	sc := NewStringTracker("abcde")
	n := sc.Skip(3)
	assert.Equal(t, 3, n)
	assert.Equal(t, 2, sc.Pos()) // 0-indexed, skipped bytes 0,1,2

	ch, ok := sc.Next()
	require.True(t, ok)
	assert.Equal(t, byte('d'), ch)
}

func TestStringTracker_SkipPastEnd(t *testing.T) {
	sc := NewStringTracker("ab")
	n := sc.Skip(5)
	assert.Equal(t, 2, n)
}

func TestStringTracker_InStringVariants(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		checkAt int // byte offset to check
		inDbl   bool
		inSgl   bool
		inBt    bool
	}{
		{"in double", `"hi"`, 1, true, false, false},
		{"in single", `'hi'`, 1, false, true, false},
		{"in backtick", "`hi`", 1, false, false, true},
		{"in code", `x + y`, 2, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewStringTracker(tt.input)
			for {
				if _, ok := sc.Next(); !ok {
					break
				}
				if sc.Pos() == tt.checkAt {
					assert.Equal(t, tt.inDbl, sc.InDoubleString())
					assert.Equal(t, tt.inSgl, sc.InSingleString())
					assert.Equal(t, tt.inBt, sc.InBacktick())
					return
				}
			}
			t.Fatal("did not reach checkAt position")
		})
	}
}

func TestStringTracker_EmptyInput(t *testing.T) {
	sc := NewStringTracker("")
	_, ok := sc.Next()
	assert.False(t, ok)
	assert.True(t, sc.InCode())
}

func TestIsInsideString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		pos    int
		inside bool
	}{
		{"code before string", `x = "hi"`, 0, false},
		{"opening quote", `x = "hi"`, 4, false}, // original: state before pos, string not yet open
		{"inside double", `x = "hi"`, 5, true},
		{"closing quote", `x = "hi"`, 7, true}, // original: state before pos, string still open
		{"after string", `x = "hi" + y`, 9, false},
		{"inside single", `x = 'hi'`, 5, true},
		{"escaped quote", `x = "h\"i"`, 6, true},
		{"after escaped quote", `x = "h\"i"`, 7, true},
		{"past end", `abc`, 10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.inside, IsInsideString(tt.input, tt.pos))
		})
	}
}

func TestFindTopLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		find  byte
		want  int
	}{
		{"simple", `a = b + c`, '=', 2},
		{"inside string", `a = "x=y" + c`, '=', 2},
		{"inside parens", `f(a=b) = c`, '=', 7},
		{"inside brackets", `a[1=2] = c`, '=', 7},
		{"not found", `a + b`, '=', -1},
		{"nested parens", `f(g(h(x))) = y`, '=', 11},
		{"pipe not or", `a | b || c`, '|', 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindTopLevel(tt.input, func(ch byte, pos int, src string) bool {
				return ch == tt.find
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindTopLevel_AssignmentNotComparison(t *testing.T) {
	// Mimics findDestructAssign logic: find = but not == != <= >= =>
	isAssign := func(ch byte, pos int, src string) bool {
		if ch != '=' {
			return false
		}
		if pos+1 < len(src) && (src[pos+1] == '=' || src[pos+1] == '>') {
			return false
		}
		if pos > 0 && (src[pos-1] == '!' || src[pos-1] == '<' || src[pos-1] == '>' || src[pos-1] == '=') {
			return false
		}
		return true
	}

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"simple assign", `x = y`, 2},
		{"skip ==", `x == y`, -1},
		{"skip !=", `x != y`, -1},
		{"skip <=", `x <= y`, -1},
		{"skip >=", `x >= y`, -1},
		{"skip =>", `x => y`, -1},
		{"assign after comparison", `x == y; z = w`, 10},
		{"assign in string ignored", `x = "a = b"`, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindTopLevel(tt.input, isAssign)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindAllTopLevel(t *testing.T) {
	// Find all pipe operators (| but not ||)
	isPipe := func(ch byte, pos int, src string) bool {
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
	}

	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{"no pipes", `a + b`, nil},
		{"one pipe", `a | b`, []int{2}},
		{"two pipes", `a | b | c`, []int{2, 6}},
		{"pipe and or", `a | b || c | d`, []int{2, 11}},
		{"pipe in string", `a | "x|y" | c`, []int{2, 10}},
		{"pipe in parens", `f(a|b) | c`, []int{7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindAllTopLevel(tt.input, isPipe)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindTopLevel_CompoundOp(t *testing.T) {
	isCompoundOp := func(op string) func(byte, int, string) bool {
		return func(ch byte, pos int, src string) bool {
			return pos+len(op) <= len(src) && src[pos:pos+len(op)] == op
		}
	}

	tests := []struct {
		name  string
		input string
		op    string
		want  int
	}{
		{"+=", `x += 1`, "+=", 2},
		{"-=", `x -= 1`, "-=", 2},
		{"+= in string", `x = "a += b"`, "+=", -1},
		{"+= in parens", `f(a += b)`, "+=", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindTopLevel(tt.input, isCompoundOp(tt.op))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBracketHelpers(t *testing.T) {
	assert.True(t, IsOpenBracket('('))
	assert.True(t, IsOpenBracket('['))
	assert.True(t, IsOpenBracket('{'))
	assert.False(t, IsOpenBracket(')'))

	assert.True(t, IsCloseBracket(')'))
	assert.True(t, IsCloseBracket(']'))
	assert.True(t, IsCloseBracket('}'))
	assert.False(t, IsCloseBracket('('))
}
