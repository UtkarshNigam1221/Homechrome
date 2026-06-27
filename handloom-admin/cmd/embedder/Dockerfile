# syntax=docker/dockerfile:1.6

# Stage 1: cross-compile the Go binary (CGO, linked against libtokenizers) on the
# NATIVE build platform, targeting arm64. Pinning the builder to arm64 made the
# CGO `go build` run under qemu emulation on amd64 CI runners — 20+ min and no
# progress output. $BUILDPLATFORM keeps the builder native (fast); GOARCH +
# aarch64 cross-gcc produce the arm64 binary the Lambda runtime needs.
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder

WORKDIR /src

# CGO toolchain + aarch64 cross-compiler. The arm64 *cross* dev packages are only
# Recommended deps (skipped by --no-install-recommends), so list them explicitly:
#   libc6-dev-arm64-cross        → arm64 libc headers (bits/libc-header-start.h)
#   libstdc++-12-dev-arm64-cross → arm64 libstdc++.so for the CGO link (-lstdc++,
#                                  pulled in by the tokenizers lib)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev pkg-config curl gcc-aarch64-linux-gnu \
    libc6-dev-arm64-cross libstdc++-12-dev-arm64-cross \
  && rm -rf /var/lib/apt/lists/*

# Pre-fetch libtokenizers (linux-arm64 prebuilt from the Go binding's release).
ARG TOKENIZERS_VERSION=v1.27.0
RUN curl -fsSL "https://github.com/daulet/tokenizers/releases/download/${TOKENIZERS_VERSION}/libtokenizers.linux-arm64.tar.gz" \
    -o /tmp/libtokenizers.tar.gz \
  && mkdir -p /tmp/libtok && tar xzf /tmp/libtokenizers.tar.gz -C /tmp/libtok \
  && cp /tmp/libtok/libtokenizers.a /usr/local/lib/libtokenizers.a

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1 GOOS=linux GOARCH=arm64
ENV CC=aarch64-linux-gnu-gcc
ENV CGO_LDFLAGS="-L/usr/local/lib"
RUN go build -ldflags="-s -w" -o /out/bootstrap ./cmd/embedder

# Stage 2: AL2023 ARM64 Lambda runtime with ORT shared lib + model assets.
FROM public.ecr.aws/lambda/provided:al2023-arm64

COPY --from=builder /out/bootstrap                                       ${LAMBDA_TASK_ROOT}/bootstrap
COPY cmd/embedder/assets/libonnxruntime.so                               ${LAMBDA_TASK_ROOT}/lib/libonnxruntime.so
COPY cmd/embedder/assets/model-int8.onnx                                 ${LAMBDA_TASK_ROOT}/model/model-int8.onnx
COPY cmd/embedder/assets/tokenizer.json                                  ${LAMBDA_TASK_ROOT}/model/tokenizer.json

ENV LD_LIBRARY_PATH=${LAMBDA_TASK_ROOT}/lib
ENV ONNXRUNTIME_SHARED_LIB_PATH=${LAMBDA_TASK_ROOT}/lib/libonnxruntime.so
ENV MODEL_PATH=${LAMBDA_TASK_ROOT}/model/model-int8.onnx
ENV TOKENIZER_PATH=${LAMBDA_TASK_ROOT}/model/tokenizer.json

ENTRYPOINT [ "/var/task/bootstrap" ]
