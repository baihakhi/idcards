# Stage 1: Builder
FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache git build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the static binary app
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /root/

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/static ./static
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]
