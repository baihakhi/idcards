FROM golang:1.22-slim

# Install dependencies for SQLite + CGO
RUN apt-get update && apt-get install -y gcc libc6-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ⛏ Compile with CGO enabled
ENV CGO_ENABLED=1
RUN go build -o app .

EXPOSE 8080

CMD ["./app"]
