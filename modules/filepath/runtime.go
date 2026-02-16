package filepathmod

import (
	"fmt"
	"path/filepath"
)

// --- filepath module ---

type Filepath struct{}

func (*Filepath) Join(extra ...interface{}) interface{} {
	parts := make([]string, len(extra))
	for i, e := range extra {
		parts[i] = fmt.Sprintf("%v", e)
	}
	return filepath.Join(parts...)
}

func (*Filepath) Base(path string) interface{} {
	return filepath.Base(path)
}

func (*Filepath) Dir(path string) interface{} {
	return filepath.Dir(path)
}

func (*Filepath) Ext(path string) interface{} {
	return filepath.Ext(path)
}

func (*Filepath) Abs(path string) interface{} {
	result, err := filepath.Abs(path)
	if err != nil {
		panic(fmt.Sprintf("filepath.abs: %v", err))
	}
	return result
}

func (*Filepath) Rel(basepath, targpath string) interface{} {
	result, err := filepath.Rel(basepath, targpath)
	if err != nil {
		panic(fmt.Sprintf("filepath.rel: %v", err))
	}
	return result
}

func (*Filepath) Glob(pattern string) interface{} {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		panic(fmt.Sprintf("filepath.glob: %v", err))
	}
	result := make([]interface{}, len(matches))
	for i, m := range matches {
		result[i] = m
	}
	return result
}

func (*Filepath) Clean(path string) interface{} {
	return filepath.Clean(path)
}

func (*Filepath) IsAbs(path string) interface{} {
	return filepath.IsAbs(path)
}

func (*Filepath) Split(path string) interface{} {
	dir, file := filepath.Split(path)
	return []interface{}{dir, file}
}

func (*Filepath) Match(pattern, name string) interface{} {
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		panic(fmt.Sprintf("filepath.match: %v", err))
	}
	return matched
}
