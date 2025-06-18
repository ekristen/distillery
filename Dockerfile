# syntax=docker/dockerfile:1.17-labs@sha256:0e0e138ab0fff10b5eaed8d6a98ef487e0ed61f77d7ce502113c6d90790ae866
FROM cgr.dev/chainguard/wolfi-base:latest@sha256:b72df108f3388c82b0638bcfbad1511d85c60593e67fb8f8a968255f7e0588df as base
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
