FROM golang:latest AS build
COPY . /go/src/build
WORKDIR /go/src/build
RUN git config --global --add safe.directory /go/src/build
RUN CGO_ENABLED=0 go build -v -x -a \
  -ldflags '-extldflags "-static"' \
  -o /user-service \
  ./internal/cmd/serve-mysql/...

FROM alpine:3.14
COPY ./sql /sql
COPY --from=build /user-service /user-service
ENTRYPOINT [ "/user-service" ]