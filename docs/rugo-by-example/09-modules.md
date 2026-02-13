# Modules

Rugo has multiple import styles, including `use` and `import`.

```ruby
use "str"
puts str.upper("hello")
```

```text
HELLO
```

```ruby
import "strings"
puts strings.to_upper("rugo")
```

```text
RUGO
```

- `use`: Rugo stdlib modules.
- `import`: Go stdlib bridge.
