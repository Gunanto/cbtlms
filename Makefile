APP=cbtlms

.PHONY: run
run:
	go run ./cmd/web

.PHONY: compose-up
compose-up:
	docker compose -f deployments/docker-compose.yml up -d

.PHONY: compose-down
compose-down:
	docker compose -f deployments/docker-compose.yml down

.PHONY: migrate-status
migrate-status:
	./scripts/migrate.sh status

.PHONY: migrate-up
migrate-up:
	./scripts/migrate.sh up

.PHONY: migrate-down
migrate-down:
	./scripts/migrate.sh down

.PHONY: migrate-reset
migrate-reset:
	./scripts/migrate.sh reset
