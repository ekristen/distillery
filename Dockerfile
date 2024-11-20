# syntax=docker/dockerfile:1.11-labs@sha256:abe0052c66a1352ebf6b2493c77459e4764af249d507c33e0d6176ac20075292
FROM cgr.dev/chainguard/wolfi-base:latest@sha256:b3dd9cf08283b959c6a0a3c833e68b2882a50129930215060154b43ae6a3e81c as base
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