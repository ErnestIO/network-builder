install:
	go install -v

build:
	go build -v ./...

lint:
	golint ./...
	go vet ./...

test:
	go test -v ./...

cover:
	go test -v ./... --cover

deps: dev-deps
	go get -u github.com/nats-io/nats
	go get -u gopkg.in/redis.v3

dev-deps:
	go get -u github.com/golang/lint/golint

clean:
	go clean
