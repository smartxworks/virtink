FROM golang:1.19-alpine

RUN apk add --no-cache bash gcc git musl-dev libaio-dev
RUN apk add --no-cache sanlock-dev --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

RUN git clone --branch=v0.24.1 --depth=1 https://github.com/kubernetes/code-generator.git $GOPATH/src/k8s.io/code-generator
RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen
RUN go install github.com/golang/mock/mockgen
