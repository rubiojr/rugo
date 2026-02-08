# fmt

String formatting using Go's `fmt.Sprintf` and `fmt.Printf`.

```ruby
use "fmt"
```

## Functions

### `fmt.sprintf(format, args...)`

Returns a formatted string.

```ruby
s = fmt.sprintf("Hello %s, you are %d years old", "Alice", 30)
# "Hello Alice, you are 30 years old"

fmt.sprintf("%05d", 42)     # "00042"
fmt.sprintf("%.2f", 3.14)   # "3.14"
fmt.sprintf("%x", 255)      # "ff"
fmt.sprintf("%q", "hello")  # "\"hello\""
```

### `fmt.printf(format, args...)`

Prints a formatted string to stdout. Returns nil.

```ruby
fmt.printf("%-10s %5d\n", "Alice", 95)
```

## Common Format Verbs

| Verb | Description |
|------|-------------|
| `%s` | String |
| `%d` | Integer |
| `%f` | Float (use `%.2f` for 2 decimal places) |
| `%x` | Hexadecimal |
| `%o` | Octal |
| `%b` | Binary |
| `%q` | Quoted string |
| `%v` | Default format |
| `%%` | Literal percent sign |
| `%05d` | Zero-padded to 5 digits |
| `%-10s` | Left-aligned, 10 chars wide |
