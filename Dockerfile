FROM golang:1.11
ADD ./ /go/src/github.com/j-fuentes/prometheus-go-service-example
WORKDIR /go/src/github.com/j-fuentes/prometheus-go-service-example
RUN make

FROM bitnami/minideb
EXPOSE 8080
WORKDIR /opt/
COPY --from=0 /go/src/github.com/j-fuentes/prometheus-go-service-example/server .
ADD ./data /opt/data
CMD ["./server"]
