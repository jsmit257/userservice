build:
	go build ./...

build-docker:
	docker-compose up --build build-serve-mysql

unit:
	go test -cover ./...

system:

vet:

fmt:
	go fmt ./...

docker:
