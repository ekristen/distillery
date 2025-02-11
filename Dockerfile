# syntax=docker/dockerfile:1.13-labs@sha256:d4250176a22a73cb8cdeb0cdcd3ea65d39baad1245f2f1dcb5eceadedd0518b8
FROM cgr.dev/chainguard/wolfi-base:latest@sha256:d00375343cbe13df4658ff32d233fd4e5dbadf0390b4b50a48ad005d85cc5b7e as base
ARG PROJECT_NAME=dist
RUN apk add --no-cache ca-certificates
RUN addgroup -S ${PROJECT_NAME} && adduser -S ${PROJECT_NAME} -G ${PROJECT_NAME}

FROM golang:1.22.4 AS build
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
