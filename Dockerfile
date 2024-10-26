# syntax=docker/dockerfile:1.10-labs@sha256:940282bab7a18daad689c238d407ad22393369ad53c6125c9c00f8be8a9da678
FROM cgr.dev/chainguard/wolfi-base:latest@sha256:3221f96f52fca0020fa6f404b0370d132403be6b3736d8dd92275ccd72129c1f as base
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