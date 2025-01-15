BINARY_NAME=main
MAIN_PATH=./main.go

build:
	golangci-lint run
	go build -o ${BINARY_NAME} ${MAIN_PATH}

run:
	golangci-lint run
	go run main.go .

run-f:
	golangci-lint run
	go run main.go . -f
