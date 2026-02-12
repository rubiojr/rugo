package testmod

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
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
	Func func() (passed bool, skipped bool, skipReason string, failReason string)
}

// rugo_test_runner executes tests and produces TAP output with optional color.
func rugo_test_runner(tests []rugoTestCase, setup, teardown, setupFile, teardownFile func() interface{}, testInstance *Test) {
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

	// Per-test timeout (0 = disabled)
	var testTimeout time.Duration
	if envTimeout := os.Getenv("RUGO_TEST_TIMEOUT"); envTimeout != "" {
		if v, err := strconv.Atoi(envTimeout); err == nil && v > 0 {
			testTimeout = time.Duration(v) * time.Second
		}
	}

	showTiming := os.Getenv("RUGO_TEST_TIMING") != ""
	showRecap := os.Getenv("RUGO_TEST_RECAP") != ""

	type failedTest struct {
		num    int
		name   string
		reason string
	}
	var failures []failedTest

	totalTests := 0
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	fmt.Println("TAP version 13")
	fmt.Printf("1..%d\n", len(tests))
	suiteStart := time.Now()

	if teardownFile != nil {
		defer teardownFile()
	}
	if setupFile != nil {
		setupFile()
	}

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

		testStart := time.Now()
		passed, skipped, skipReason, failReason := rugo_run_test_with_timeout(t.Func, testTimeout)
		testElapsed := time.Since(testStart)

		if teardown != nil {
			teardown()
		}

		// Cleanup per-test temp directory
		os.RemoveAll(tmpDir)
		testInstance.TmpDir = ""

		timingSuffix := ""
		if showTiming {
			timingSuffix = fmt.Sprintf(" (%s)", rugo_format_duration(testElapsed))
		}

		if skipped {
			fmt.Printf("%sok%s %d - %s # SKIP %s%s\n", colorSkip, colorReset, testNum, t.Name, skipReason, timingSuffix)
			totalSkipped++
		} else if passed {
			fmt.Printf("%sok%s %d - %s%s\n", colorOK, colorReset, testNum, t.Name, timingSuffix)
			totalPassed++
		} else {
			fmt.Printf("%snot ok%s %d - %s%s\n", colorFail, colorReset, testNum, t.Name, timingSuffix)
			totalFailed++
			if showRecap && failReason != "" {
				failures = append(failures, failedTest{num: testNum, name: t.Name, reason: failReason})
			}
		}
	}

	// Print recap of failures before the summary
	if len(failures) > 0 {
		fmt.Fprintf(os.Stderr, "\n%s--- Failed tests ---%s\n", colorFail, colorReset)
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "\n  %snot ok %d - %s%s\n", colorFail, f.num, f.name, colorReset)
			for _, line := range strings.Split(f.reason, "\n") {
				fmt.Fprintf(os.Stderr, "    %s\n", line)
			}
		}
	}

	timingTotal := ""
	if showTiming {
		timingTotal = fmt.Sprintf(" in %s", rugo_format_duration(time.Since(suiteStart)))
	}

	if totalFailed > 0 {
		fmt.Fprintf(os.Stderr, "\n%d tests, %d passed, %s%d failed%s, %d skipped%s\n", totalTests, totalPassed, colorFail, totalFailed, colorReset, totalSkipped, timingTotal)
	} else {
		fmt.Fprintf(os.Stderr, "\n%d tests, %s%d passed%s, %d failed, %d skipped%s\n", totalTests, colorOK, totalPassed, colorReset, totalFailed, totalSkipped, timingTotal)
	}
	if totalFailed > 0 {
		os.Exit(1)
	}
}

// rugo_run_test_with_timeout runs a test function with an optional timeout.
// If timeout is 0, the test runs without a deadline.
func rugo_run_test_with_timeout(fn func() (bool, bool, string, string), timeout time.Duration) (passed bool, skipped bool, skipReason string, failReason string) {
	if timeout == 0 {
		return fn()
	}

	type testResult struct {
		passed     bool
		skipped    bool
		skipReason string
		failReason string
	}

	done := make(chan testResult, 1)
	go func() {
		p, s, sr, fr := fn()
		done <- testResult{p, s, sr, fr}
	}()

	select {
	case res := <-done:
		return res.passed, res.skipped, res.skipReason, res.failReason
	case <-time.After(timeout):
		failColor := "\033[31m"
		failReset := "\033[0m"
		if os.Getenv("NO_COLOR") != "" {
			failColor = ""
			failReset = ""
		}
		msg := fmt.Sprintf("test timed out after %v", timeout)
		fmt.Fprintf(os.Stderr, "  %sFAIL%s: %s\n", failColor, failReset, msg)
		return false, false, "", msg
	}
}

// rugo_format_duration formats a duration in a human-friendly way.
func rugo_format_duration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
