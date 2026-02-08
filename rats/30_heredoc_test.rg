# RATS: Heredoc multiline strings
use "test"

rats "basic interpolating heredoc"
  result = test.run("rugo run rats/fixtures/heredoc_basic.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Hello World")
  test.assert_eq(lines[1], "Welcome to Rugo")
end

rats "squiggly heredoc strips common indent"
  result = test.run("rugo run rats/fixtures/heredoc_squiggly.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "<h1>Hello World</h1>")
  test.assert_eq(lines[1], "<p>Welcome</p>")
end

rats "raw heredoc preserves literal content"
  result = test.run("rugo run rats/fixtures/heredoc_raw.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], 'Hello #{name}')
  test.assert_eq(lines[1], "No interpolation")
end

rats "raw squiggly heredoc strips indent without interpolation"
  result = test.run("rugo run rats/fixtures/heredoc_raw_squiggly.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "def foo")
  test.assert_eq(lines[1], '  puts "hello"')
  test.assert_eq(lines[2], "end")
end

rats "empty heredoc produces empty string"
  result = test.run("rugo run rats/fixtures/heredoc_empty.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "><")
end

rats "heredoc preserves quotes backslashes and hash chars"
  result = test.run("rugo run rats/fixtures/heredoc_special_chars.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], 'Line with "double quotes"')
  test.assert_eq(lines[1], 'And backslash: C:\path')
  test.assert_eq(lines[2], "Has # not a comment")
end

rats "closing delimiter can be indented"
  result = test.run("rugo run rats/fixtures/heredoc_indented_delim.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello")
end

rats "multiple heredocs in one file"
  result = test.run("rugo run rats/fixtures/heredoc_multiple.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Hello")
  test.assert_eq(lines[1], "World")
end

rats "heredoc inside a function"
  result = test.run("rugo run rats/fixtures/heredoc_in_function.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Hello Rugo")
  test.assert_eq(lines[1], "Welcome!")
end

rats "unterminated heredoc reports error"
  result = test.run("rugo run rats/fixtures/err_heredoc_unterminated.rg")
  test.assert_eq(result["status"], 1)
  test.assert_contains(result["output"], "unterminated heredoc")
end

rats "heredoc emits valid Go"
  result = test.run("rugo emit rats/fixtures/heredoc_basic.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "fmt.Sprintf")
end

rats "heredoc compiles to native binary"
  test.run("rugo build -o " + test.tmpdir() + "/heredoc_test rats/fixtures/heredoc_basic.rg")
  result = test.run(test.tmpdir() + "/heredoc_test")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Hello World")
  test.assert_eq(lines[1], "Welcome to Rugo")
end
