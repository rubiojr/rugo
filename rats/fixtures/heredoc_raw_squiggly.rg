# Raw squiggly heredoc: no interpolation + strip indent
msg = <<~'CODE'
    def foo
      puts "hello"
    end
CODE
puts msg
