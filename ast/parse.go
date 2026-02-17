package ast

import (
	"fmt"
	"os"
	"strings"

	"github.com/rubiojr/rugo/parser"
	"github.com/rubiojr/rugo/preprocess"
	"modernc.org/scanner"
)

// Compiler provides Rugo source parsing into typed AST nodes.
type Compiler struct{}

// ParseFile reads a Rugo source file and parses it into a Program AST.
func (c *Compiler) ParseFile(filename string) (*Program, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}
	return c.ParseSource(string(src), filename)
}

// ParseSource parses raw Rugo source code into a Program AST.
// The name parameter is used for error messages.
func (c *Compiler) ParseSource(source, name string) (*Program, error) {
	rawSource := source

	cleaned, heredocLineMap, err := preprocess.ExpandHeredocs(source)
	if err != nil {
		return nil, fmt.Errorf("%s:%w", name, err)
	}

	cleaned, err = preprocess.StripComments(cleaned)
	if err != nil {
		return nil, fmt.Errorf("%s:%w", name, err)
	}

	var structLineMap []int
	var structInfos []preprocess.StructInfo
	cleaned, structLineMap, structInfos = preprocess.ExpandStructDefs(cleaned)

	userFuncs := preprocess.ScanFuncDefs(cleaned)

	var lineMap []int
	cleaned, lineMap, err = preprocess.Preprocess(cleaned, userFuncs)
	if err != nil {
		return nil, fmt.Errorf("%s:%w", name, err)
	}

	// Compose line maps: preprocess → struct → heredoc → original source.
	if structLineMap != nil && lineMap != nil {
		for i, ppLine := range lineMap {
			if ppLine > 0 && ppLine <= len(structLineMap) {
				lineMap[i] = structLineMap[ppLine-1]
			}
		}
	} else if structLineMap != nil {
		lineMap = structLineMap
	}

	if heredocLineMap != nil && lineMap != nil {
		for i, ppLine := range lineMap {
			if ppLine > 0 && ppLine <= len(heredocLineMap) {
				lineMap[i] = heredocLineMap[ppLine-1]
			}
		}
	} else if heredocLineMap != nil {
		lineMap = heredocLineMap
	}

	if !strings.HasSuffix(cleaned, "\n") {
		cleaned += "\n"
	}

	p := &parser.Parser{}
	flatAST, err := p.Parse(name, []byte(cleaned))
	if err != nil {
		return nil, firstParseError(err)
	}

	prog, err := WalkWithLineMap(p, flatAST, lineMap)
	if err != nil {
		return nil, fmt.Errorf("%s: internal error: %w", name, err)
	}

	prog.SourceFile = name
	prog.RawSource = rawSource
	prog.Structs = structInfos
	return prog, nil
}

// firstParseError extracts the first error from a parser error list.
func firstParseError(err error) error {
	if el, ok := err.(scanner.ErrList); ok && len(el) > 0 {
		return fmt.Errorf("%s", el[0])
	}
	return err
}
