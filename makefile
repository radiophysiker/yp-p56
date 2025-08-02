GOOSE := go run github.com/pressly/goose/v3/cmd/goose
DB_DSN ?= postgres://gopher:gopher@localhost:5432/gophermart?sslmode=disable

.PHONY: migrate-up migrate-down migrate-status

migrate-up:
	$(GOOSE) -dir ./migrations postgres "$(DB_DSN)" up

migrate-down:
	$(GOOSE) -dir ./migrations postgres "$(DB_DSN)" down

migrate-status:
	$(GOOSE) -dir ./migrations postgres "$(DB_DSN)" status
