# RATS: Test cli module (commands, flags, dispatch, help)
import "test"

rats "cli dispatch calls correct handler"
  result = test.run("rugo run rats/fixtures/cli_greet.rg hello")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, World!")
end

rats "cli string flag works"
  result = test.run("rugo run rats/fixtures/cli_greet.rg hello -n Rugo")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, Rugo!")
end

rats "cli long flag works"
  result = test.run("rugo run rats/fixtures/cli_greet.rg hello --name Developer")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, Developer!")
end

rats "cli bool flag works"
  result = test.run("rugo run rats/fixtures/cli_greet.rg goodbye -l")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "GOODBYE!")
end

rats "cli bool flag default is false"
  result = test.run("rugo run rats/fixtures/cli_greet.rg goodbye")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Goodbye!")
end

rats "cli no command shows help"
  result = test.run("rugo run rats/fixtures/cli_greet.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "greet")
  test.assert_contains(result["output"], "Commands:")
  test.assert_contains(result["output"], "hello")
end

rats "cli --help shows help"
  result = test.run("rugo run rats/fixtures/cli_greet.rg --help")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "A friendly greeter")
end

rats "cli --version shows version"
  result = test.run("rugo run rats/fixtures/cli_greet.rg --version")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "1.0.0")
end

rats "cli command --help shows command help"
  result = test.run("rugo run rats/fixtures/cli_greet.rg hello --help")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Say hello")
  test.assert_contains(result["output"], "--name")
end

rats "cli subcommands with colon notation"
  result = test.run("rugo run rats/fixtures/cli_subcommands.rg db:migrate")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "migrating")
end

rats "cli subcommand dispatch"
  result = test.run("rugo run rats/fixtures/cli_subcommands.rg db:seed")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "seeding")
end

rats "cli space-separated subcommands"
  result = test.run("rugo run rats/fixtures/cli_subcmds.rg db migrate")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "migrating")
end

rats "cli space-separated subcommand dispatch"
  result = test.run("rugo run rats/fixtures/cli_subcmds.rg server start")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "starting")
end

rats "cli space-separated subcommand help"
  result = test.run("rugo run rats/fixtures/cli_subcmds.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "db migrate")
  test.assert_contains(result["output"], "server start")
end

rats "cli positional args passed to handler"
  result = test.run("rugo run rats/fixtures/cli_positional.rg echo foo bar baz")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "foo")
  test.assert_eq(lines[1], "bar")
  test.assert_eq(lines[2], "baz")
end

rats "cli unknown flag exits with error"
  result = test.run("rugo run rats/fixtures/cli_greet.rg hello --bad")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "unknown flag")
end

rats "cli compiles to native binary"
  test.run("rugo build -o " + test.tmpdir() + "/cli_test rats/fixtures/cli_greet.rg")
  result = test.run(test.tmpdir() + "/cli_test hello -n Binary")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, Binary!")
end
