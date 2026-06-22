# Stage 1: builder
FROM golang:1.25-alpine AS builder

ARG VERSION=dev

WORKDIR /src

# Copy go.mod first for layer caching; go.sum is optional (stdlib-only module).
COPY go.mod go.sum* ./
RUN go mod download

# Copy the rest of the source tree.
COPY . .

# Compute build metadata at image build time.
RUN GIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) && \
    BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    CGO_ENABLED=0 go build \
        -ldflags "-X main.Version=${VERSION} -X main.GitHash=${GIT_HASH} -X main.BuildDate=${BUILD_DATE}" \
        -o /out/eve-realm-mcp \
        ./cmd/eve-realm-mcp

# Stage 2: runtime
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/eve-realm-mcp /usr/local/bin/eve-realm-mcp

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/eve-realm-mcp"]
