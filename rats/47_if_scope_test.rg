use "test"

def if_else_scope
  if true
    bar = "bar"
  else
    bar = "stuff"
  end
  return bar
end

def if_only_scope
  if true
    x = 42
  end
  return x
end

def elsif_scope
  val = 0
  if false
    val = 1
  elsif true
    val = 2
  else
    val = 3
  end
  return val
end

def nested_if_scope
  if true
    if true
      inner = "deep"
    end
  end
  return inner
end

def new_var_in_all_branches(flag)
  if flag
    msg = "yes"
  else
    msg = "no"
  end
  return msg
end

rats "variable assigned in if/else is visible after end"
  test.assert_eq(if_else_scope(), "bar")
end

rats "variable assigned in if-only is visible after end"
  test.assert_eq(if_only_scope(), 42)
end

rats "variable assigned in elsif branch"
  test.assert_eq(elsif_scope(), 2)
end

rats "variable assigned in nested if is visible"
  test.assert_eq(nested_if_scope(), "deep")
end

rats "variable assigned in both branches works"
  test.assert_eq(new_var_in_all_branches(true), "yes")
  test.assert_eq(new_var_in_all_branches(false), "no")
end
