# syntax=docker/dockerfile:1.12-labs@sha256:5a2914b8a3ae788a4b8874f80dddde9fdf932e1d224fab8bab669bd18f251f9a
FROM cgr.dev/chainguard/wolfi-base:latest@sha256:d653e81239ff979cf810ca6f706d1bcf56d6994da75f37b4702327a3c2696900 as base
ARG PROJECT_NAME=distillery
RUN apk add --no-cache ca-certificates
RUN addgroup -S ${PROJECT_NAME} && adduser -S ${PROJECT_NAME} -G ${PROJECT_NAME}

FROM ghcr.io/acorn-io/images-mirror/golang:1.21@sha256:856073656d1a517517792e6cdd2f7a5ef080d3ca2dff33e518c8412f140fdd2d AS build
ARG PROJECT_NAME=distillery
COPY / /src
WORKDIR /src
RUN \
  --mount=type=cache,target=/go/pkg \
  --mount=type=cache,target=/root/.cache/go-build \
  go build -o bin/${PROJECT_NAME} main.go

FROM base AS goreleaser
ARG PROJECT_NAME=distillery
COPY ${PROJECT_NAME} /usr/local/bin/${PROJECT_NAME}
USER ${PROJECT_NAME}

FROM base
ARG PROJECT_NAME=distillery
COPY --from=build /src/bin/${PROJECT_NAME} /usr/local/bin/${PROJECT_NAME}
USER ${PROJECT_NAME}