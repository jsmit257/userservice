.PHONY: build
build:
	go build ./...

.PHONY: unit
unit:
	go test -cover ./...

.PHONY: mysql-migration
mysql-migration:
	docker-compose up --build --force-recreate --remove-orphans mysql-migration

.PHONY: mysql-test
mysql-test:
	docker-compose up --build --force-recreate --remove-orphans -d mysql-test

.PHONY: mysql-persist
mysql-persist:
	docker-compose up --build --force-recreate --remove-orphans -d mysql-persist

.PHONY: serve-mysql
serve-mysql:
	docker-compose up --build --force-recreate --remove-orphans -d serve-mysql

.PHONY: system-test
system-test: unit docker-down mysql-test serve-mysql
	./bin/test-integration

.PHONY: vet
vet:

.PHONY: docker-down
docker-down:
	docker-compose down
