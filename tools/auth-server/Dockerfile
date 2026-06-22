FROM golang:1.26.4-alpine AS base

RUN adduser --uid 1000 --disabled-password user && \
    apk add -U --no-cache ca-certificates

WORKDIR /app

COPY tools/auth-server ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    go mod download && go mod verify

RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -v -o main ./

# --------------------------------------------------------
FROM scratch AS release

COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base --chmod=700 --chown=1000:1000 /app/main main

USER user
EXPOSE 8900
CMD ["./main"]
