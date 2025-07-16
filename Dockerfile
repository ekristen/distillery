# syntax=docker/dockerfile:1.17-labs@sha256:9187104f31e3a002a8a6a3209ea1f937fb7486c093cbbde1e14b0fa0d7e4f1b5
FROM cgr.dev/chainguard/wolfi-base:latest@sha256:aaf69a3f822e069db4be5047e9a0c5d25dd78396d9e0ea5e718b8e542978981e as base
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
