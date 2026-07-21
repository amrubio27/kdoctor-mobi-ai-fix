# syntax=docker/dockerfile:1

# Stage 1: Build the kdoctor Go binary.
FROM golang:1.23-alpine AS builder
WORKDIR /build

# Download Go modules first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 go build -o kdoctor ./cmd/kdoctor

# Stage 2: Download the detekt CLI.
FROM eclipse-temurin:17-jre-alpine AS detekt-downloader
ARG DETEKT_VERSION=1.23.8
RUN apk add --no-cache curl ca-certificates && \
    curl -L -o /detekt-cli.jar \
    "https://github.com/detekt/detekt/releases/download/v${DETEKT_VERSION}/detekt-cli-${DETEKT_VERSION}-all.jar" && \
    apk del curl ca-certificates

# Stage 3: Runtime image.
FROM eclipse-temurin:17-jre-alpine
LABEL org.opencontainers.image.title="kdoctor" \
      org.opencontainers.image.description="Android / KMP / CMP health scanner" \
      org.opencontainers.image.licenses="MIT"

COPY --from=builder /build/kdoctor /usr/local/bin/kdoctor
COPY --from=detekt-downloader /detekt-cli.jar /usr/local/lib/detekt-cli.jar

# The image ships detekt at /usr/local/lib/detekt-cli.jar.
# Run scans with: --detekt-bin /usr/local/lib/detekt-cli.jar
ENTRYPOINT ["kdoctor"]
CMD ["--help"]
