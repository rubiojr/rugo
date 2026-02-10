# user_repo.rg — Lightweight OOP pattern wrapping sqlite
use "sqlite"

struct UserRepo
  conn
end

def UserRepo.add(name, age)
  sqlite.exec(self.conn, "INSERT INTO users (name, age) VALUES (?, ?)", name, age)
end

def UserRepo.find(id)
  return sqlite.query_row(self.conn, "SELECT * FROM users WHERE id = ?", id)
end

def UserRepo.all()
  return sqlite.query(self.conn, "SELECT * FROM users ORDER BY name")
end

def UserRepo.count()
  return sqlite.query_val(self.conn, "SELECT COUNT(*) FROM users")
end
