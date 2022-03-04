VERSION 0.6
FROM golang:1.17
WORKDIR /vault-plugin-secrets-tailscale

deps:
    COPY go.mod go.sum ./
    RUN go mod download
    SAVE ARTIFACT go.mod AS LOCAL go.mod
    SAVE ARTIFACT go.sum AS LOCAL go.sum

build:
    FROM +deps
    COPY *.go .
    COPY --dir ./cmd .
    RUN CGO_ENABLED=0 go build -o bin/vault-plugin-secrets-tailscale cmd/tailscale/main.go
    SAVE ARTIFACT bin/vault-plugin-secrets-tailscale /tailscale AS LOCAL bin/vault-plugin-secrets-tailscale

test:
    FROM +deps
    COPY *.go .
    ARG TEST_TAILSCALE_TAILNET=bloominlabs
    RUN --secret TEST_TAILSCALE_TOKEN TEST_TAILSCALE_TAILNET=$TEST_TAILSCALE_TAILNET CGO_ENABLED=0 go test github.com/bloominlabs/vault-plugin-secrets-tailscale

dev:
  BUILD +build
  LOCALLY
  RUN bash ./scripts/dev.sh

all:
  BUILD +build
  BUILD +test
