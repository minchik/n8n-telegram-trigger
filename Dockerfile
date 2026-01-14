FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0
ENV GOOS=linux

# Install ca-certificates for SSL/TLS support
RUN apk --no-cache add ca-certificates

RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -a -installsuffix cgo -ldflags="-w -s" -o n8n-telegram-trigger .

FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy ca-certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /app/n8n-telegram-trigger /app/n8n-telegram-trigger

USER 10001:10001

WORKDIR /app

ENTRYPOINT ["/app/n8n-telegram-trigger"]
