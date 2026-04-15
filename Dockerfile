FROM golang:1.25.1-alpine AS builder

RUN apk add --no-cache git ca-certificates

RUN adduser -D -g '' appuser

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY assets/ assets/
COPY cmd/ cmd/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o api ./cmd/api/main.go

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /app/api /api

USER appuser

EXPOSE 8090

CMD ["/api"]