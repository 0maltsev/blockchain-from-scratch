build:
	go build -o ./bin/blockchain-from-scratch

run: build
	./bin/blockchain-from-scratch

test:
	go test ./...