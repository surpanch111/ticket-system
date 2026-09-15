# Stage 1: Build the static executable
FROM golang:1.22-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ticket-system main.go

# Stage 2: Final minimal container
FROM alpine:3.20
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/ticket-system .

EXPOSE 8080
CMD ["./ticket-system"]