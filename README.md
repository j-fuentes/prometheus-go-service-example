# Example Golang service with prometheus metrics

## Build and execute

### With docker:

```
docker build -t myservice .
docker run -ti -p 8080:8080 myservice
```

### Native (requires Go):

```
make
./server
```

or just

```
make serve
```
