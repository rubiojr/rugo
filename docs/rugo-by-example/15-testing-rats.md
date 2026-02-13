# Testing with RATS

RATS uses `rats` blocks and the `test` module.

```ruby
use "test"

rats "adds numbers"
  sum = 2 + 3
  test.assert_eq(sum, 5)
end
```

```text
```

Run all tests:

```bash
rugo rats --timing --recap
```

```text
1 passed, 0 failed
```
