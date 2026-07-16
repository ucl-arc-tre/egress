FROM golang:1.26.4-alpine AS base

RUN adduser --uid 1000 --disabled-password user && \
    apk add -U --no-cache ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    go mod download && go mod verify

COPY internal ./internal
COPY cmd ./cmd
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -v -o main cmd/main.go

# --------------------------------------------------------
FROM base AS dev

# Expects the repo to be mounted in under /repo
WORKDIR /repo
RUN go install github.com/air-verse/air@v1.65.3
CMD ["air", "--build.cmd", "go build -o bin/main cmd/main.go", "--build.entrypoint", "./bin/main"]

# --------------------------------------------------------
FROM scratch AS release

COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base --chmod=700 --chown=1000:1000 /app/main main

ENV GIN_MODE="release"
ENV DEBUG="false"
ENV HTTP_PORT="8000"
ENV HTTPS_PORT="8443"

USER user
CMD ["./main"]
