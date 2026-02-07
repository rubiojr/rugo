# RATS: Test escape sequences in string literals
import "test"

rats "hex escape \\x1b produces ANSI escape"
  result = test.run("rugo run rats/fixtures/escape_hex.rg")
  test.assert_eq(result["status"], 0)
  # Output should contain the ESC character (0x1b) followed by ANSI codes
  result = test.run("rugo run rats/fixtures/escape_hex.rg | cat -v")
  test.assert_contains(result["output"], "^[[32mgreen^[[0m")
end

rats "octal escape \\033 produces ANSI escape"
  result = test.run("rugo run rats/fixtures/escape_octal.rg")
  test.assert_eq(result["status"], 0)
  result = test.run("rugo run rats/fixtures/escape_octal.rg | cat -v")
  test.assert_contains(result["output"], "^[[31mred^[[0m")
end

rats "hex escapes produce correct ASCII characters"
  result = test.run("rugo run rats/fixtures/escape_hex_ascii.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello")
end

rats "octal escapes produce correct ASCII characters"
  result = test.run("rugo run rats/fixtures/escape_octal_ascii.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello")
end

rats "hex escapes work with string interpolation"
  result = test.run("rugo run rats/fixtures/escape_interpolation.rg | cat -v")
  test.assert_contains(result["output"], "^[[32mgreen^[[0m")
end

rats "classic escapes still work (\\n \\t)"
  result = test.run("rugo run rats/fixtures/escape_classic.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "a\tb\tc")
  test.assert_eq(lines[1], "line1")
  test.assert_eq(lines[2], "line2")
end

rats "hex escape in emit output produces valid Go"
  result = test.run("rugo emit rats/fixtures/escape_hex.rg")
  test.assert_eq(result["status"], 0)
  # With raw strings, we can directly check for the literal \x1b pattern
  test.assert_contains(result["output"], '\x1b')
end

rats "octal escape in emit output produces valid Go"
  result = test.run("rugo emit rats/fixtures/escape_octal.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], '\x1b')
end

rats "escape sequences compile to native binary"
  test.run("rugo build -o " + test.tmpdir() + "/escape_test rats/fixtures/escape_hex_ascii.rg")
  result = test.run(test.tmpdir() + "/escape_test")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello")
end
