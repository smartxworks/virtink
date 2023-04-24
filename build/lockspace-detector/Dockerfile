FROM golang:1.19-alpine AS builder

RUN apk add --no-cache gcc musl-dev libaio-dev
RUN apk add --no-cache sanlock-dev --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN --mount=type=cache,target=/root/.cache/go-build go build -a cmd/lockspace-detector/main.go

FROM alpine

RUN apk add --no-cache sanlock --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/

COPY --from=builder /workspace/main /usr/bin/lockspace-detector
ENTRYPOINT ["lockspace-detector"]
