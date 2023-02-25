FROM golang:latest as build
RUN ls -lh
WORKDIR /go/src/jsmit257
CMD CGO_ENABLED=0 go build -v -x -a \
  -ldflags '-extldflags "-static"' \
  -o ./internal/cmd/serve-mysql/user-service \
  ./internal/cmd/serve-mysql/main.go

FROM alpine:3.14 as deploy
COPY ./internal/cmd/serve-mysql/user-service /user-service
