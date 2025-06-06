# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.1-alpine3.21 AS build

ENV GO111MODULE=on
ARG MAKE_TARGET=go-build
# Arguments used to stamp the binary
ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

COPY Makefile go.mod go.sum /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/
COPY version.mk common.mk /go/src/github.com/open-edge-platform/virtual-edge-node/
COPY cmd/ /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/cmd/
COPY pkg/ /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/
COPY vendor/ /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/vendor/

RUN apk add --no-cache make=4.4.1-r2

WORKDIR /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-simulator
RUN CGO_ENABLED=0 DOCKER_LABEL_REPO_URL=${REPO_URL} DOCKER_LABEL_VERSION=${VERSION} DOCKER_LABEL_REVISION=${REVISION} DOCKER_LABEL_BUILD_DATE=${BUILD_DATE} make ${MAKE_TARGET}

FROM alpine:3.21@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c

ARG REPO_URL
ARG VERSION
ARG REVISION
ARG BUILD_DATE

LABEL org.opencontainers.image.version=${VERSION} \
      org.opencontainers.image.source=${REPO_URL} \
      org.opencontainers.image.revision=${REVISION} \
      org.opencontainers.image.created=${BUILD_DATE}

USER nobody

COPY --from=build /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/out/ensim/server/main /usr/local/bin/ensim-server
COPY --from=build /go/src/github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/out/ensim/client/main /usr/local/bin/ensim-client

ENTRYPOINT ["ensim-server"]
