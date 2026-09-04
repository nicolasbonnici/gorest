FROM golang:1.25.1-alpine AS builder

# ── root module: migrate + user CLIs ─────────────────────────────────────────
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o migrate ./migrations/cmd/migrate/
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o user     ./cmd/user/

# ── example module: api ───────────────────────────────────────────────────────
WORKDIR /build/examples/basic-api
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o api .

FROM alpine:3.24

WORKDIR /app
COPY --from=builder /build/migrate         .
COPY --from=builder /build/user            .
COPY --from=builder /build/examples/basic-api/api        .
COPY --from=builder /build/examples/basic-api/gorest.yaml .

EXPOSE 3000
CMD ["./api"]
