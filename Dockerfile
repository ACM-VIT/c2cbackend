# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM scratch
# Copy the binary
COPY --from=builder /app/main /main

# Create certs directory
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ # (needed for TLS)
# Add CockroachDB root certificate
ADD https://cockroachlabs.cloud/clusters/21a98a1a-f5f3-4b5c-b50b-f3b886d16bd3/cert /root/.postgresql/root.crt

# Set env for Postgres driver
ENV SSLROOTCERT=/root/.postgresql/root.crt

CMD ["/main"]
