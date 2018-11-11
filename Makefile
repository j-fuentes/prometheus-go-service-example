all: build

build: get
	go build -o server .

get:
	go get ./...

vet:
	go vet ./...

lint: vet
	golint

serve:
	go run ./main.go
