//go:build ignore

package osmod

// --- os module ---

type OS struct{}

func (*OS) Exec(command string) interface{} {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("os.exec failed: %v", err))
	}
	return strings.TrimRight(string(out), "\n")
}

func (*OS) Exit(code int) interface{} {
	os.Exit(code)
	return nil
}
