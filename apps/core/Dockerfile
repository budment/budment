FROM golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/blaster ./cmd/blaster

# Runtime Stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/blaster /usr/local/bin/blaster

ENTRYPOINT ["blaster"]