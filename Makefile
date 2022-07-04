.PHONY: test
test:
	go test ./...

.PHONY: install
install:
	go get ./...

.PHONY: clean
clean:
	rm -rf ./bin

broadcast-poll: ./handlers/broadcast-poll/main.go
	go build -o ./bin/broadcast-poll ./handlers/broadcast-poll

create-poll: ./handlers/create-poll/main.go
	go build -o ./bin/create-poll ./handlers/create-poll

get-poll: ./handlers/get-poll/main.go
	go build -o ./bin/get-poll ./handlers/get-poll

.PHONY: handlers
handlers:
	GOOS=linux GOARCH=amd64 $(MAKE) broadcast-poll
	GOOS=linux GOARCH=amd64 $(MAKE) create-poll
	GOOS=linux GOARCH=amd64 $(MAKE) get-poll

.PHONY: build
build: clean handlers

.PHONY: run
run: build
	sam local start-api