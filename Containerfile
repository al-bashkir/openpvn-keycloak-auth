# Containerfile
#
# Build the openvpn-keycloak-sso Go binary in a container and export it
# to the local ./build directory:
#
#   podman build -f Containerfile -o type=local,dest=build .
#
# Output:
#   build/openvpn-keycloak-sso
#
# Optional: embed version metadata (matches Makefile conventions):
#
#   podman build -f Containerfile \
#     --build-arg VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
#     --build-arg COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" \
#     --build-arg BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
#     -o type=local,dest=build .

FROM docker.io/golang:1.24-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . ./

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

ARG GOOS=linux
ARG GOARCH=amd64

RUN mkdir -p /out && \
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
      -trimpath \
      -buildvcs=false \
      -ldflags="-s -w -X 'main.version=$VERSION' -X 'main.commit=$COMMIT' -X 'main.buildDate=$BUILD_DATE'" \
      -o /out/openvpn-keycloak-sso \
      ./cmd/openvpn-keycloak-sso

# Minimal artifact stage (used by podman --output type=local,dest=...)
FROM scratch AS artifact

COPY --from=build /out/openvpn-keycloak-sso /openvpn-keycloak-sso
