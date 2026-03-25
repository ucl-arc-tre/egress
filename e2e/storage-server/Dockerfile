FROM golang:1.26.0-alpine AS base

RUN adduser --uid 1000 --disabled-password user && \
    apk add -U --no-cache ca-certificates && \
    mkdir -p /tmp/storage-root

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    go mod download && go mod verify

COPY pkg ./pkg
COPY e2e/storage-server ./e2e/storage-server
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -v -o main ./e2e/storage-server

# --------------------------------------------------------
FROM scratch AS release

COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base --chmod=700 --chown=1000:1000 /app/main main

# /tmp/storage-root is the hardcoded storage root (see main.go). The directory
# is baked in here so the path exists in the scratch image; in Kubernetes it is
# overlaid by an emptyDir volume mount, providing a writable filesystem
COPY --from=base /tmp /tmp

USER user
EXPOSE 8800
CMD ["./main"]
