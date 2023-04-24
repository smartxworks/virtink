FROM golang:1.19-alpine AS builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -a cmd/virt-controller/main.go

FROM alpine

ARG PRERUNNER_IMAGE
ENV PRERUNNER_IMAGE=$PRERUNNER_IMAGE
ARG DETECTOR_IMAGE
ENV DETECTOR_IMAGE=$DETECTOR_IMAGE
ARG ATTACHER_IMAGE
ENV ATTACHER_IMAGE=$ATTACHER_IMAGE
ARG INITIALIZER_IMAGE
ENV INITIALIZER_IMAGE=$INITIALIZER_IMAGE

COPY --from=builder /workspace/main /usr/bin/virt-controller
ENTRYPOINT ["virt-controller"]
