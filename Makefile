PKGS := $(shell go list ./...)

.PHONY: test up down
test:
	docker compose -f docker-compose-test.yaml up --build --abort-on-container-exit

up:
	docker compose up --build

down:
	docker compose down

