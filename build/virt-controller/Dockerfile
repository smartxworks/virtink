FROM golang:1.23.4-alpine3.21 AS builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -a cmd/virt-controller/main.go

FROM alpine:3.21.0

ARG PRERUNNER_IMAGE
ENV PRERUNNER_IMAGE=$PRERUNNER_IMAGE

COPY --from=builder /workspace/main /usr/bin/virt-controller
ENTRYPOINT ["virt-controller"]
