FROM golang:1.25.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o api .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/api .
COPY gorest.yaml ./

EXPOSE 3000
CMD ["./api"]
