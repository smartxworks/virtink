FROM golang:1.17-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN --mount=type=cache,target=/root/.cache/go-build go build -a cmd/kubrid-prerunner/main.go

FROM alpine

RUN apk add --no-cache curl screen dnsmasq cdrkit iptables

RUN curl -sLo /usr/bin/cloud-hypervisor https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v24.0/cloud-hypervisor-static && chmod +x /usr/bin/cloud-hypervisor
RUN mkdir /var/lib/cloud-hypervisor && curl -sLo /var/lib/cloud-hypervisor/hypervisor-fw https://github.com/cloud-hypervisor/rust-hypervisor-firmware/releases/download/0.4.0/hypervisor-fw
RUN curl -sLo /usr/bin/ch-remote https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v24.0/ch-remote-static && chmod +x /usr/bin/ch-remote

COPY --from=builder /workspace/main /usr/bin/kubrid-prerunner
COPY build/kubrid-prerunner/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]

ADD build/kubrid-prerunner/kubrid-init-volume.sh /usr/bin/kubrid-init-volume
