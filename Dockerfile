# syntax=docker/dockerfile:1.15-labs@sha256:8378c88c56f2a6c038705487ce1e447c61c48557cd6a76aea4d53e255304260a
FROM cgr.dev/chainguard/wolfi-base:latest@sha256:6dbd3e4b965942b30546dfbe6582d19e8023704d63f09e3205c08b4aa9adc8cf as base
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
