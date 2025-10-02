FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o restgen ./cmd/server

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/restgen .

ENV PORT=3000
ENV DB_URL=postgres://postgres:postgres@db:5432/mydb?sslmode=disable
ENV JWT_SECRET=supersecret

EXPOSE 3000
CMD ["./restgen"]
