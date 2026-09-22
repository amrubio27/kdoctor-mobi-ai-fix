# syntax=docker/dockerfile:1

# Stage 1: Build the kdoctor binaries.
FROM golang:1.23-alpine AS builder
WORKDIR /build

# Download Go modules first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
# VERSION is stamped into the binary; without it `kdoctor --version` says "dev".
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o kdoctor ./cmd/kdoctor && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o kdoctor-mcp ./cmd/kdoctor-mcp

# Stage 2: Download the detekt CLI.
#
# Pulled from Maven Central rather than GitHub Releases because Maven publishes
# a .sha256 next to the artifact, so the download can be verified instead of
# trusted. Keep DETEKT_VERSION and DETEKT_SHA256 in sync with
# internal/core/detektrunner/provision.go.
FROM eclipse-temurin:17-jre-alpine AS detekt-downloader
ARG DETEKT_VERSION=1.23.8
ARG DETEKT_SHA256=3afe89a11120303c73c9bdda3d8fe558dd9070a6937d27819ddc04b275381245
ARG COMPOSE_RULES_VERSION=0.4.22
RUN apk add --no-cache curl ca-certificates && \
    curl -fsSL -o /detekt-cli.jar \
      "https://repo1.maven.org/maven2/io/gitlab/arturbosch/detekt/detekt-cli/${DETEKT_VERSION}/detekt-cli-${DETEKT_VERSION}-all.jar" && \
    echo "${DETEKT_SHA256}  /detekt-cli.jar" | sha256sum -c - && \
    mkdir -p /compose-rules && \
    curl -fsSL -o "/compose-rules/detekt-${COMPOSE_RULES_VERSION}.jar" \
      "https://repo1.maven.org/maven2/io/nlopez/compose/rules/detekt/${COMPOSE_RULES_VERSION}/detekt-${COMPOSE_RULES_VERSION}.jar" && \
    curl -fsSL -o "/compose-rules/common-${COMPOSE_RULES_VERSION}.jar" \
      "https://repo1.maven.org/maven2/io/nlopez/compose/rules/common/${COMPOSE_RULES_VERSION}/common-${COMPOSE_RULES_VERSION}.jar" && \
    apk del curl ca-certificates

# Stage 3: Runtime image.
FROM eclipse-temurin:17-jre-alpine
LABEL org.opencontainers.image.title="kdoctor" \
      org.opencontainers.image.description="Android / KMP / CMP health scanner" \
      org.opencontainers.image.source="https://github.com/amrubio27/kdoctor-mobi-ai-fix" \
      org.opencontainers.image.licenses="MIT"

COPY --from=builder /build/kdoctor /usr/local/bin/kdoctor
COPY --from=builder /build/kdoctor-mcp /usr/local/bin/kdoctor-mcp
COPY --from=detekt-downloader /detekt-cli.jar /usr/local/lib/detekt-cli.jar
COPY --from=detekt-downloader /compose-rules /usr/local/lib/compose-rules

# Everything is baked in, so `docker run … kdoctor scan` works with no flags and
# no network. Previously the image shipped detekt but the user still had to pass
# --detekt-bin by hand, and a scan would silently try to fetch it.
ENV KDOCTOR_DETEKT_JAR=/usr/local/lib/detekt-cli.jar \
    KDOCTOR_COMPOSE_PLUGINS=/usr/local/lib/compose-rules/detekt-0.4.22.jar:/usr/local/lib/compose-rules/common-0.4.22.jar \
    KDOCTOR_NO_DOWNLOAD=1

WORKDIR /project
ENTRYPOINT ["kdoctor"]
CMD ["--help"]
