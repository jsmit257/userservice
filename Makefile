.PHONY: build
build:
	go build ./...

.PHONY: unit
unit:
	go test -cover ./...

.PHONY: mysql-migration
mysql-migration:
	docker-compose up --build --remove-orphans mysql-migration

.PHONY: mysql-test
mysql-test:
	docker-compose up --build --remove-orphans -d mysql-test

.PHONY: mysql-persist
mysql-persist:
	docker-compose up --build --remove-orphans -d mysql-persist

.PHONY: serve-mysql
serve-mysql:
	docker-compose up --build --remove-orphans -d serve-mysql

.PHONY: system-test
system-test: unit down mysql-test serve-mysql
	echo "sleeping while mail connects"
	sleep 1s
	./bin/test-integration

.PHONY: web-test
web-test:
	docker-compose down -t5 us-web-test
	docker-compose up --build --remove-orphans -d us-web-test

.PHONY: vet
vet:

.PHONY: down
down:
	docker-compose down -t5

.PHONY: deploy
deploy: # no hard dependency on `system-test` for mow
	docker-compose build mysql-test
	docker-compose build mysql-migration
	docker-compose build serve-mysql
	docker-compose build us-web-test
	docker tag jsmit257/us-db-mysql-test:latest jsmit257/us-db-mysql-test:lkg
	docker tag jsmit257/us-db-mysql-mig:latest jsmit257/us-db-mysql-mig:lkg
	docker tag jsmit257/us-srv-mysql:latest jsmit257/us-srv-mysql:lkg
	docker tag jsmit257/us-web-test:latest jsmit257/us-web-test:lkg

.PHONY: push
push: # just docker, not git
	docker push jsmit257/us-db-mysql-test:lkg
	docker push jsmit257/us-db-mysql-mig:lkg
	docker push jsmit257/us-srv-mysql:lkg
	docker push jsmit257/us-web-test:lkg
