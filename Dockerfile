FROM golang:1.25.1-alpine AS builder

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY ./api ./api
COPY ./cmd ./cmd
COPY ./internal ./internal

ARG VERSION=DEVELOPMENT_VERSION
# Build
RUN CGO_ENABLED=0 GOOS=linux go generate ./internal/webserver
RUN CGO_ENABLED=0 GOOS=linux go build -C ./cmd/openem-ingestor-service/ -v -o /app/ingestor  -ldflags="-s -w  -X 'main.version=${VERSION}'"

FROM ubuntu:24.04

RUN apt-get update && \
    apt-get install -y ca-certificates 

COPY --from=builder /app/ingestor /app/ingestor

EXPOSE 8080

WORKDIR /app

RUN chmod -R a+rwx /app

# Run
CMD ["./ingestor"]