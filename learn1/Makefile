BINARY_NAME=main

build:
	golangci-lint run
	go build -o ${BINARY_NAME} ./cmd/main.go

run:
	golangci-lint run
	go build -o ${BINARY_NAME} ./cmd/main.go
	./${BINARY_NAME}
	rm ${BINARY_NAME}