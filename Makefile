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
	echo "sleeping while things connect"
	sleep 2s
	./bin/test-integration

.PHONY: web
web:
	rmdir nginx/www/js nginx/www/css
	docker-compose up --build --remove-orphans -d us-web

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
	docker-compose build us-web
	docker tag jsmit257/us-db-mysql-test:latest jsmit257/us-db-mysql-test:lkg
	docker tag jsmit257/us-db-mysql-mig:latest jsmit257/us-db-mysql-mig:lkg
	docker tag jsmit257/us-srv-mysql:latest jsmit257/us-srv-mysql:lkg
	docker tag jsmit257/us-web:latest jsmit257/us-web:lkg
	git tag -f stable

.PHONY: push
push:
	docker push jsmit257/us-db-mysql-test:lkg
	docker push jsmit257/us-db-mysql-mig:lkg
	docker push jsmit257/us-srv-mysql:lkg
	docker push jsmit257/us-web:lkg
	git push --force origin stable:stable

