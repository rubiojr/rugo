# ast

Parse and inspect Rugo source files.

```ruby
use "ast"
```

> **Requires:** Go toolchain (`go`) on PATH. The ast module compiles a
> helper binary to parse Rugo source.

## parse_file

Parses a `.rugo` file and returns a program hash.

```ruby
prog = ast.parse_file("lib.rugo")
puts prog["source_file"]

for stmt in prog["statements"]
  if stmt["type"] == "def"
    puts stmt["name"]
  end
end
```

The returned hash has these keys:

| Key | Type | Description |
|-----|------|-------------|
| `source_file` | String | Path to the parsed file |
| `raw_source` | String | Original source text |
| `statements` | Array | Array of statement hashes |
| `structs` | Array | Array of struct hashes |

Panics if the file cannot be read or has syntax errors.

## parse_source

Parses a Rugo source string and returns the same program hash as
`parse_file`. The second argument is a name used in error messages.

```ruby
source = <<~SRC
  def greet(name)
    return "hello " + name
  end
SRC

prog = ast.parse_source(source, "example")
stmt = prog["statements"][0]
puts stmt["name"]     # greet
puts stmt["params"]   # [name]
```

## source_lines

Extracts the raw source lines for a statement from a program. Returns an
array of strings.

```ruby
source = <<~SRC
  x = 10
  def greet(name)
    return "hello " + name
  end
SRC

prog = ast.parse_source(source, "example")
for stmt in prog["statements"]
  lines = ast.source_lines(prog, stmt)
  for line in lines
    puts line
  end
end
```

## Statement Hashes

Every statement hash has `type`, `line`, and `end_line` keys. Additional keys
depend on the type:

| Type | Extra Keys |
|------|------------|
| `def` | `name`, `params` (array), `body` (array of statements) |
| `assign` | `target` |
| `if` | `body`, `elsif` (array), `else_body` |
| `while` | `body` |
| `for` | `var`, `index_var` (optional), `body` |
| `return` | — |
| `break` | — |
| `next` | — |
| `use` | `module` |
| `import` | `package`, `alias` (optional) |
| `require` | `path`, `alias` (optional), `with` (optional array) |
| `expr` | `expr` (expression hash) |
| `test` | `name`, `body` |
| `bench` | `name`, `body` |

## Struct Hashes

Each entry in `prog["structs"]` has:

| Key | Type | Description |
|-----|------|-------------|
| `name` | String | Struct name |
| `fields` | Array | Field name strings |
| `line` | Integer | Source line number |
