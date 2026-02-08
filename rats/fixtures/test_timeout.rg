use "test"

rats "passes quickly"
  test.assert_eq(1, 1)
end

rats "times out"
  `sleep 10`
  test.assert_eq(1, 1)
end
