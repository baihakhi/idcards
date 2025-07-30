APP_NAME = idcard-app

run:
	go run ./cmd

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
