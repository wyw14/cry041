$env:DATABASE_URL = if ($env:DATABASE_URL) { $env:DATABASE_URL } else { 'postgres://gate:gate@localhost:5432/gate?sslmode=disable' }
go run ./cmd/server
