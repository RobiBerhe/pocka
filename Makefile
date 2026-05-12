# Makefile for Pocka Expense Tracker

.PHONY: up down restart build logs ps clean db-shell ent-gen help

# Default target
help:
	@echo "Available commands:"
	@echo "  make up         - Start all services in background"
	@echo "  make down       - Stop and remove all containers"
	@echo "  make build      - Build or rebuild services"
	@echo "  make restart    - Restart all services"
	@echo "  make logs       - Tail all service logs"
	@echo "  make ps         - Show status of containers"
	@echo "  make db-shell   - Access PostgreSQL shell"
	@echo "  make clean      - Remove containers, images, and volumes"
	@echo "  make ent-gen    - Generate Ent ORM code"

# Docker commands
up:
	docker-compose up -d --build

down:
	docker-compose down

restart:
	docker-compose restart

build:
	docker-compose build

logs:
	docker-compose logs -f

ps:
	docker-compose ps

clean:
	docker-compose down --rmi all --volumes --remove-orphans

db-shell:
	docker-compose exec db psql -U pocka_user -d pocka_db

# Development commands
ent-gen:
	go generate ./ent
