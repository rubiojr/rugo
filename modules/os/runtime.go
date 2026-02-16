package osmod

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

var stdinReader *bufio.Reader

func (*OS) ReadLine(prompt string) interface{} {
	if prompt != "" {
		fmt.Print(prompt)
	}
	if stdinReader == nil {
		stdinReader = bufio.NewReader(os.Stdin)
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return strings.TrimRight(line, "\r\n")
	}
	return strings.TrimRight(line, "\r\n")
}

func (*OS) Getenv(key string) interface{} {
	return os.Getenv(key)
}

func (*OS) Setenv(key, value string) interface{} {
	if err := os.Setenv(key, value); err != nil {
		panic(fmt.Sprintf("os.setenv failed: %v", err))
	}
	return nil
}

func (*OS) Cwd() interface{} {
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("os.cwd failed: %v", err))
	}
	return dir
}

func (*OS) Chdir(path string) interface{} {
	if err := os.Chdir(path); err != nil {
		panic(fmt.Sprintf("os.chdir failed: %v", err))
	}
	return nil
}

func (*OS) Hostname() interface{} {
	h, err := os.Hostname()
	if err != nil {
		panic(fmt.Sprintf("os.hostname failed: %v", err))
	}
	return h
}

func (*OS) ReadFile(path string) interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("os.read_file failed: %v", err))
	}
	return string(data)
}

func (*OS) WriteFile(path, content string) interface{} {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		panic(fmt.Sprintf("os.write_file failed: %v", err))
	}
	return nil
}

func (*OS) Remove(path string) interface{} {
	if err := os.RemoveAll(path); err != nil {
		panic(fmt.Sprintf("os.remove failed: %v", err))
	}
	return nil
}

func (*OS) Mkdir(path string) interface{} {
	if err := os.MkdirAll(path, 0755); err != nil {
		panic(fmt.Sprintf("os.mkdir failed: %v", err))
	}
	return nil
}

func (*OS) Rename(oldpath, newpath string) interface{} {
	if err := os.Rename(oldpath, newpath); err != nil {
		panic(fmt.Sprintf("os.rename failed: %v", err))
	}
	return nil
}

func (*OS) Glob(pattern string) interface{} {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		panic(fmt.Sprintf("os.glob failed: %v", err))
	}
	result := make([]interface{}, len(matches))
	for i, m := range matches {
		result[i] = m
	}
	return result
}

func (*OS) TmpDir() interface{} {
	return os.TempDir()
}

func (*OS) Args() interface{} {
	args := os.Args[1:]
	result := make([]interface{}, len(args))
	for i, a := range args {
		result[i] = a
	}
	return result
}

func (*OS) Pid() interface{} {
	return os.Getpid()
}

func (*OS) Symlink(oldname, newname string) interface{} {
	if err := os.Symlink(oldname, newname); err != nil {
		panic(fmt.Sprintf("os.symlink failed: %v", err))
	}
	return nil
}

func (*OS) Readlink(name string) interface{} {
	target, err := os.Readlink(name)
	if err != nil {
		panic(fmt.Sprintf("os.readlink failed: %v", err))
	}
	return target
}
