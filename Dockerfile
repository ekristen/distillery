# syntax=docker/dockerfile:1.14-labs@sha256:16aafd1d331afa5f314579cc4faf5ceb0e3ae15812d2a9da926c14175da73dc7
FROM cgr.dev/chainguard/wolfi-base:latest@sha256:5ec50de5d68fc25ca132976c4f4c29e2763749210aef0e3811281fb3a6a9031b as base
ARG PROJECT_NAME=dist
RUN apk add --no-cache ca-certificates
RUN addgroup -S ${PROJECT_NAME} && adduser -S ${PROJECT_NAME} -G ${PROJECT_NAME}

FROM golang:1.24.0@sha256:4546829ecda4404596cf5c9d8936488283910a3564ffc8fe4f32b33ddaeff239 AS build
ARG PROJECT_NAME=dist
COPY / /src
WORKDIR /src
RUN \
  --mount=type=cache,target=/go/pkg \
  --mount=type=cache,target=/root/.cache/go-build \
  go build -o bin/${PROJECT_NAME} main.go

FROM base AS goreleaser
ARG PROJECT_NAME=dist
COPY ${PROJECT_NAME} /usr/local/bin/${PROJECT_NAME}

FROM base
ARG PROJECT_NAME=dist
COPY --from=build /src/bin/${PROJECT_NAME} /usr/local/bin/${PROJECT_NAME}
