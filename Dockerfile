FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /signalstream ./cmd/signalstream

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /signalstream .

EXPOSE 8080

ENTRYPOINT ["./signalstream"]
