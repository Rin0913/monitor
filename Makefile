.PHONY: test up down
up:
	docker compose up --build

test:
	docker compose -f docker-compose-test.yaml up --build --abort-on-container-exit

down:
	docker compose down

