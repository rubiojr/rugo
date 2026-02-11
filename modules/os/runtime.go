package osmod

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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

func (*OS) FileExists(path string) interface{} {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (*OS) IsDir(path string) interface{} {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
