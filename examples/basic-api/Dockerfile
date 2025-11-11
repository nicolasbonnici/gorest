FROM golang:1.25.1-alpine AS builder

WORKDIR /build/examples/basic-api

COPY examples/basic-api/go.mod examples/basic-api/go.sum ./
COPY . /build/

RUN go mod download
RUN go build -o api .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /build/examples/basic-api/api .
COPY --from=builder /build/examples/basic-api/gorest.yaml ./

EXPOSE 3000
CMD ["./api"]
