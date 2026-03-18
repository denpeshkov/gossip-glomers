# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25
ARG APP_VERSION=devel

# ========================================
# Build Stage
# ========================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build

ENV GOCACHE=/.cache/go-build
ENV GOMODCACHE=/.cache/go-mod

WORKDIR /src

ARG TARGETARCH
ARG TARGETOS

RUN --mount=type=cache,target=${GOMODCACHE} \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOARCH=${TARGETARCH} GOOS=${TARGETOS} \
    go build \
    -trimpath \
    -ldflags "-s -w -X github.com/denpeshkov/go-template/buildinfo.Version=${VERSION}" \
    -o /bin/go-template \
    ./cmd/go-template

# ========================================
# Production Stage
# ========================================
FROM gcr.io/distroless/static:nonroot

COPY --from=build /bin/go-template .
EXPOSE 8080
ENTRYPOINT [./go-template"]
