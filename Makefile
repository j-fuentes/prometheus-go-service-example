all: build

build:
	go build -o server .

vet:
	go vet ./...

lint: vet
	golint

serve:
	go run ./main.go
