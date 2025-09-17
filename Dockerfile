# syntax=docker/dockerfile:1.18-labs@sha256:79cdc14e1c220efb546ad14a8ebc816e3277cd72d27195ced5bebdd226dd1025
FROM cgr.dev/chainguard/wolfi-base:latest@sha256:0e09bcd548cf2dfb9a3fd40af1a7389aa8c16b428de4e8f72b085f015694ce3d as base
ARG PROJECT_NAME=dist
RUN apk add --no-cache ca-certificates
RUN addgroup -S ${PROJECT_NAME} && adduser -S ${PROJECT_NAME} -G ${PROJECT_NAME}

FROM golang:1.25.1@sha256:8305f5fa8ea63c7b5bc85bd223ccc62941f852318ebfbd22f53bbd0b358c07e1 AS build
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
