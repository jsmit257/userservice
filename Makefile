.PHONY: build
build:
	go build ./...

.PHONY: unit
unit:
	go test -cover ./...

.PHONY: compile-serve-mysql
compile-serve-mysql: build
	docker-compose up --build --force-recreate build-serve-mysql

.PHONY: package-serve-mysql
package-serve-mysql: compile-serve-mysql
	docker-compose up --build --force-recreate package-serve-mysql

.PHONY: test-serve-mysql
test-serve-mysql: build
	docker-compose down
	docker-compose up --build --force-recreate test-serve-mysql

system:

vet:

fmt:
	go fmt ./...

docker:
