IMAGE_NAME?=josefuentes/quiz
IMAGE_TAG?=latest
IMAGE_FULL_NAME=$(IMAGE_NAME):$(IMAGE_TAG)

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
	go run .

push:
	docker build -t $(IMAGE_FULL_NAME) .
	docker push $(IMAGE_FULL_NAME)
