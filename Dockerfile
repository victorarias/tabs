FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o tabs-server cmd/tabs-server/main.go

FROM alpine:3.19
COPY --from=builder /build/tabs-server /usr/local/bin/
EXPOSE 8080
CMD ["tabs-server"]
