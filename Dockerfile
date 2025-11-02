FROM golang:1.25.1-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o gorest ./cmd/gorest/main.go

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/gorest .

ENV PORT=3000
ENV DB_URL=postgres://postgres:postgres@db:5432/mydb?sslmode=disable
# JWT_SECRET must be provided at runtime via environment variable
# Example: docker run -e JWT_SECRET=$JWT_SECRET ...

EXPOSE 3000
CMD ["./gorest"]
