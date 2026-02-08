struct Dog
  name
  breed
end

def Dog.bark()
  return self.name + " says woof!"
end

def Dog.describe()
  return self.name + " (" + self.breed + ")"
end

def Dog.rename(new_name)
  self.name = new_name
end
