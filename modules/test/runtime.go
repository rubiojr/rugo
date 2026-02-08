package testmod

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// --- test module ---

type rugoTestSkip string
type rugoTestFail string

type Test struct {
	TmpDir string
}

func (t *Test) Tmpdir() interface{} {
	return t.TmpDir
}

// WriteFile writes content to a file, creating it with 0644 permissions.
func (*Test) WriteFile(path, content string) interface{} {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		panic(rugoTestFail(fmt.Sprintf("write_file failed: %v", err)))
	}
	return nil
}

// Run executes a command and returns a hash with status, output, and lines.
func (*Test) Run(command string) interface{} {
	cmd := exec.Command("sh", "-c", command)
	// Ensure child output has no ANSI codes so string matching works reliably.
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	output := strings.TrimRight(string(out), "\n")

	status := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			status = exitErr.ExitCode()
		} else {
			status = -1
		}
	}

	// Split output into lines
	var lines []interface{}
	if output != "" {
		for _, line := range strings.Split(output, "\n") {
			lines = append(lines, interface{}(line))
		}
	}

	result := map[interface{}]interface{}{
		"status": status,
		"output": output,
		"lines":  lines,
	}
	return result
}

func (*Test) AssertEq(actual, expected interface{}) interface{} {
	if actual != expected {
		panic(rugoTestFail(fmt.Sprintf("assert_eq failed\n  expected: %v\n       got: %v", expected, actual)))
	}
	return nil
}

func (*Test) AssertNeq(actual, expected interface{}) interface{} {
	if actual == expected {
		panic(rugoTestFail(fmt.Sprintf("assert_neq failed: both values are %v", actual)))
	}
	return nil
}

func (*Test) AssertTrue(val interface{}) interface{} {
	if !rugo_to_bool(val) {
		panic(rugoTestFail(fmt.Sprintf("assert_true failed: got %v", val)))
	}
	return nil
}

func (*Test) AssertFalse(val interface{}) interface{} {
	if rugo_to_bool(val) {
		panic(rugoTestFail(fmt.Sprintf("assert_false failed: got %v", val)))
	}
	return nil
}

func (*Test) AssertContains(s, substr interface{}) interface{} {
	ss := rugo_to_string(s)
	sub := rugo_to_string(substr)
	if !strings.Contains(ss, sub) {
		panic(rugoTestFail(fmt.Sprintf("assert_contains failed\n  string: %q\n  substr: %q", ss, sub)))
	}
	return nil
}

func (*Test) AssertNil(val interface{}) interface{} {
	if val != nil {
		panic(rugoTestFail(fmt.Sprintf("assert_nil failed: got %v (%T)", val, val)))
	}
	return nil
}

func (*Test) Fail(msg interface{}) interface{} {
	panic(rugoTestFail(rugo_to_string(msg)))
}

func (*Test) Skip(reason interface{}) interface{} {
	panic(rugoTestSkip(rugo_to_string(reason)))
}

// Used in generated test programs, not directly in this package.
var _ = rugo_test_runner

// rugoTestCase describes a single test for the runner.
type rugoTestCase struct {
	Name string
	Func func() (passed bool, skipped bool, skipReason string)
}

// rugo_test_runner executes tests and produces TAP output with optional color.
func rugo_test_runner(tests []rugoTestCase, setup, teardown func() interface{}, testInstance *Test) {
	colorOK := "\033[32m"
	colorFail := "\033[31m"
	colorSkip := "\033[33m"
	colorReset := "\033[0m"
	if os.Getenv("NO_COLOR") != "" {
		colorOK = ""
		colorFail = ""
		colorSkip = ""
		colorReset = ""
	}

	totalTests := 0
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	fmt.Println("TAP version 13")
	fmt.Printf("1..%d\n", len(tests))

	for i, t := range tests {
		testNum := i + 1
		totalTests++

		// Create per-test temp directory
		tmpDir, tmpErr := os.MkdirTemp("", "rats-*")
		if tmpErr != nil {
			fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", tmpErr)
			os.Exit(1)
		}
		testInstance.TmpDir = tmpDir

		if setup != nil {
			setup()
		}
		passed, skipped, skipReason := t.Func()
		if teardown != nil {
			teardown()
		}

		// Cleanup per-test temp directory
		os.RemoveAll(tmpDir)
		testInstance.TmpDir = ""

		if skipped {
			fmt.Printf("%sok%s %d - %s # SKIP %s\n", colorSkip, colorReset, testNum, t.Name, skipReason)
			totalSkipped++
		} else if passed {
			fmt.Printf("%sok%s %d - %s\n", colorOK, colorReset, testNum, t.Name)
			totalPassed++
		} else {
			fmt.Printf("%snot ok%s %d - %s\n", colorFail, colorReset, testNum, t.Name)
			totalFailed++
		}
	}

	if totalFailed > 0 {
		fmt.Fprintf(os.Stderr, "\n%d tests, %d passed, %s%d failed%s, %d skipped\n", totalTests, totalPassed, colorFail, totalFailed, colorReset, totalSkipped)
	} else {
		fmt.Fprintf(os.Stderr, "\n%d tests, %s%d passed%s, %d failed, %d skipped\n", totalTests, colorOK, totalPassed, colorReset, totalFailed, totalSkipped)
	}
	if totalFailed > 0 {
		os.Exit(1)
	}
}
