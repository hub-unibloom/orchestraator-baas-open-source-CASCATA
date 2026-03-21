# Makefile para O Cascata v1.0.0.0

.PHONY: up down restart logs build fmt

# Iniciar o ambiente completo (Orquestrador + DB + Cache)
up:
	docker compose up -d

# Parar o ambiente
down:
	docker compose down

# Reiniciar o orquestrador (re-build se necessário)
restart:
	docker compose up -d --build cascata-orchestrator

# Ver logs do orquestrador
logs:
	docker compose logs -f cascata-orchestrator

# Re-build manual da imagem do backend
build:
	docker compose build

# Formatar o código Go
fmt:
	go fmt ./...
