FROM golang:1.13 AS builder
WORKDIR /app
ADD go.* /app
ADD *.go /app
RUN go test -c

FROM fermayo/containerd:debian
WORKDIR /app
COPY entrypoint.sh /app/entrypoint.sh
COPY --from=builder /app/example.test /app/example.test
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/example.test"]
