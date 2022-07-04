.PHONY: test
test:
	go test ./...

.PHONY: install
install:
	go get ./...
	go install github.com/githubnemo/CompileDaemon

.PHONY: clean
clean:
	rm -rf ./bin

aggregate-poll-votes: ./handlers/aggregate-poll-votes/main.go
	go build -o ./bin/aggregate-poll-votes ./handlers/aggregate-poll-votes

broadcast-poll: ./handlers/broadcast-poll/main.go
	go build -o ./bin/broadcast-poll ./handlers/broadcast-poll

create-poll: ./handlers/create-poll/main.go
	go build -o ./bin/create-poll ./handlers/create-poll

get-poll: ./handlers/get-poll/main.go
	go build -o ./bin/get-poll ./handlers/get-poll

submit-vote: ./handlers/submit-vote/main.go
	go build -o ./bin/submit-vote ./handlers/submit-vote

.PHONY: handlers
handlers:
	GOOS=linux GOARCH=amd64 $(MAKE) aggregate-poll-votes
	GOOS=linux GOARCH=amd64 $(MAKE) broadcast-poll
	GOOS=linux GOARCH=amd64 $(MAKE) create-poll
	GOOS=linux GOARCH=amd64 $(MAKE) get-poll
	GOOS=linux GOARCH=amd64 $(MAKE) submit-vote

.PHONY: watch
watch:
	CompileDaemon -build="make handlers"

.PHONY: build
build: clean handlers

.PHONY: run
run: build
	sam local start-api