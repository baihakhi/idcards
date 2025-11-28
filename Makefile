APP_NAME = idcard

run:
	powershell -Command "$$env:APP_NAME='$(APP_NAME)'; go run ./cmd"

run-dev:
	set ENV=dev && go run ./cmd

run-prod:
	set ENV=prod && go run ./cmd

build:
	docker build -t $(APP_NAME) .

up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

restart:
	make down && make up

clean:
	docker system prune -f
