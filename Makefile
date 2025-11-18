APP_NAME=order-service-sample

.PHONY: build up app worker down logs ps clean migrate

# Build image (without running)
build:
	docker compose build

# Run full stack (HTTP + worker + dependencies)
all:
	docker compose up --build all

# Run only HTTP server
app:
	docker compose up --build app

# Run only worker
worker:
	docker compose up --build worker

# Stop and remove all containers
down:
	docker compose down

# Show container status
ps:
	docker compose ps

# Show logs (all services)
logs:
	docker compose logs -f --tail=100

# Clean up volumes (use carefully)
clean:
	docker compose down -v
	docker system prune -f

# Run database migration inside the app container
migrate:
	cat migrations.sql | docker compose exec -T db psql -U admin -d ecommerce

rollback:
	cat rollback.sql | docker compose exec -T db psql -U admin -d ecommerce

test:
	TEST_DATABASE_DSN=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable go test ./...
