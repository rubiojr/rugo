package evalmod

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubiojr/rugo/compiler"
)

// --- eval module ---

type Eval struct{}

// checkGo verifies the Go toolchain is available.
func checkGo() {
	if _, err := exec.LookPath("go"); err != nil {
		panic("eval: Go toolchain required but \"go\" not found in PATH. Install Go from https://go.dev/dl/")
	}
}

// Run compiles and runs a Rugo source string, returning a result hash
// with keys: "status" (int), "output" (string), "lines" (array).
func (*Eval) Run(source string) interface{} {
	checkGo()

	tmpDir, err := os.MkdirTemp("", "rugo-eval-*")
	if err != nil {
		panic(fmt.Sprintf("eval.run: creating temp dir: %v", err))
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "eval.rugo")
	if err := os.WriteFile(srcFile, []byte(source), 0644); err != nil {
		panic(fmt.Sprintf("eval.run: writing source: %v", err))
	}

	c := &compiler.Compiler{BaseDir: tmpDir}
	result, err := c.RunCapture(srcFile)
	if err != nil {
		return errorToHash(err)
	}

	return capturedToHash(result)
}

// File compiles and runs a Rugo source file, returning a result hash.
// Extra arguments after the file path are passed to the compiled program.
func (*Eval) File(path string, extra ...interface{}) interface{} {
	checkGo()

	// Convert interface{} args to strings.
	args := make([]string, len(extra))
	for i, v := range extra {
		args[i] = fmt.Sprintf("%v", v)
	}

	c := &compiler.Compiler{}
	result, err := c.RunCapture(path, args...)
	if err != nil {
		return errorToHash(err)
	}

	return capturedToHash(result)
}

// capturedToHash converts a CapturedOutput to the standard result hash format
// matching test.run() for easy migration.
func capturedToHash(result *compiler.CapturedOutput) map[interface{}]interface{} {
	var lines []interface{}
	if result.Output != "" {
		for _, line := range strings.Split(result.Output, "\n") {
			lines = append(lines, interface{}(line))
		}
	}

	return map[interface{}]interface{}{
		"status": result.ExitCode,
		"output": result.Output,
		"lines":  lines,
	}
}

// errorToHash converts a compile/build error to the standard result hash,
// matching what "rugo run" outputs on failure (exit 1 + "error: " prefix).
func errorToHash(err error) map[interface{}]interface{} {
	msg := strings.TrimRight("error: "+err.Error(), "\n")
	var lines []interface{}
	for _, line := range strings.Split(msg, "\n") {
		lines = append(lines, interface{}(line))
	}
	return map[interface{}]interface{}{
		"status": 1,
		"output": msg,
		"lines":  lines,
	}
}
