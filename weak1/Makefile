BINARY_NAME=main
MAIN_PATH=./cmd/main.go

build:
	golangci-lint run
	go build -o ${BINARY_NAME} ${MAIN_PATH}

run:
	golangci-lint run
	go build -o ${BINARY_NAME} ${MAIN_PATH}
	./${BINARY_NAME}
	rm ${BINARY_NAME}