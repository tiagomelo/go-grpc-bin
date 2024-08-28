# Stage 1: Build the Go binary
FROM golang:1.23.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the Go binary for the default platform architecture
RUN CGO_ENABLED=0 go build -o /go-grpc-bin cmd/main.go

# Stage 2: Create a minimal image to run the binary
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /go-grpc-bin .

ENTRYPOINT ["./go-grpc-bin"]
