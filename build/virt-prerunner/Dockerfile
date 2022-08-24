FROM golang:1.17-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN --mount=type=cache,target=/root/.cache/go-build go build -a cmd/virt-prerunner/main.go

FROM alpine

RUN apk add --no-cache tini curl screen dnsmasq cdrkit iptables

RUN set -eux; \
    mkdir /var/lib/cloud-hypervisor; \
    case "$(uname -m)" in \
        'x86_64') \
            curl -sLo /usr/bin/cloud-hypervisor https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v26.0/cloud-hypervisor-static; \
            curl -sLo /usr/bin/ch-remote https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v26.0/ch-remote-static; \
            curl -sLo /var/lib/cloud-hypervisor/hypervisor-fw https://github.com/cloud-hypervisor/rust-hypervisor-firmware/releases/download/0.4.0/hypervisor-fw; \
            ;; \
        'aarch64') \
            curl -sLo /usr/bin/cloud-hypervisor https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v26.0/cloud-hypervisor-static-aarch64; \
            curl -sLo /usr/bin/ch-remote https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v26.0/ch-remote-static-aarch64; \
            curl -sLo /var/lib/cloud-hypervisor/CLOUDHV_EFI.fd https://github.com/smartxworks/cloud-hypervisor-edk2-builder/releases/download/20220706/CLOUDHV_EFI.fd; \
            ;; \
        *) echo >&2 "error: unsupported architecture '$(uname -m)'"; exit 1 ;; \
    esac; \
    chmod +x /usr/bin/cloud-hypervisor; \
    chmod +x /usr/bin/ch-remote

COPY --from=builder /workspace/main /usr/bin/virt-prerunner
COPY build/virt-prerunner/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/sbin/tini", "-g", "--", "/entrypoint.sh"]

ADD build/virt-prerunner/virt-init-volume.sh /usr/bin/virt-init-volume
