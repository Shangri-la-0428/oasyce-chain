# Stage 1: Build the oasyced binary
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags "-X github.com/cosmos/cosmos-sdk/version.Name=oasyce \
    -X github.com/cosmos/cosmos-sdk/version.AppName=oasyced" \
    -o /app/build/oasyced ./cmd/oasyced

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates jq bash

COPY --from=builder /app/build/oasyced /usr/local/bin/oasyced

# P2P, RPC, REST API, gRPC
EXPOSE 26656 26657 1317 9090

VOLUME /root/.oasyced

ENTRYPOINT ["oasyced", "start"]
