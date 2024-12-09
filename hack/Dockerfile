FROM golang:1.23.4

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

RUN git clone --branch=v0.24.1 --depth=1 https://github.com/kubernetes/code-generator.git $GOPATH/src/k8s.io/code-generator
RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen
RUN go install github.com/golang/mock/mockgen
