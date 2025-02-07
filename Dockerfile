FROM golang:1.23 AS builder

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
COPY configs/schemas /app/schemas

ARG VERSION=1.2.3
# Build
RUN CGO_ENABLED=0 GOOS=linux go generate ./internal/webserver
RUN CGO_ENABLED=0 GOOS=linux go build -C ./cmd/openem-ingestor-service/ -v -o /app/ingestor  -ldflags="-s -w  -X 'main.version=${VERSION}'"

FROM alpine
COPY --from=builder /app/ingestor /app/ingestor
COPY --from=builder /app/schemas /app/schemas

EXPOSE 8080
WORKDIR /app
# Run
CMD ["./ingestor"]