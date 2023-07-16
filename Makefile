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
test-serve-mysql: docker-down build
	docker-compose up --build --force-recreate test-serve-mysql

.PHONY: system
system: docker-down package-serve-mysql
	docker-compose up --build --force-recreate schema
	docker-compose up serve-mysql &
	sleep 2s
	-cd ./tests/system; go test -v ./user/...
	-curl localhost:3000/metrics

vet:

fmt:
	go fmt ./...

docker:

.PHONY: docker-down
docker-down:
	docker-compose down