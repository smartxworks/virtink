FROM golang:1.19-alpine AS builder

RUN apk add --no-cache gcc musl-dev libaio-dev
RUN apk add --no-cache sanlock-dev --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN --mount=type=cache,target=/root/.cache/go-build go build -a cmd/virt-prerunner/main.go

FROM alpine

RUN apk add --no-cache curl screen dnsmasq cdrkit iptables iproute2 qemu-virtiofsd dpkg util-linux tini nmap-ncat
RUN apk add --no-cache sanlock --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/

RUN set -eux; \
    mkdir /var/lib/cloud-hypervisor; \
    case "$(uname -m)" in \
        'x86_64') \
            curl -sLo /usr/bin/cloud-hypervisor https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v30.0/cloud-hypervisor-static; \
            curl -sLo /usr/bin/ch-remote https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v30.0/ch-remote-static; \
            curl -sLo /var/lib/cloud-hypervisor/hypervisor-fw https://github.com/cloud-hypervisor/rust-hypervisor-firmware/releases/download/0.4.0/hypervisor-fw; \
            ;; \
        'aarch64') \
            curl -sLo /usr/bin/cloud-hypervisor https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v30.0/cloud-hypervisor-static-aarch64; \
            curl -sLo /usr/bin/ch-remote https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v30.0/ch-remote-static-aarch64; \
            curl -sLo /var/lib/cloud-hypervisor/CLOUDHV_EFI.fd https://github.com/smartxworks/cloud-hypervisor-edk2-builder/releases/download/20220706/CLOUDHV_EFI.fd; \
            ;; \
        *) echo >&2 "error: unsupported architecture '$(uname -m)'"; exit 1 ;; \
    esac; \
    chmod +x /usr/bin/cloud-hypervisor; \
    chmod +x /usr/bin/ch-remote

COPY --from=builder /workspace/main /usr/bin/virt-prerunner

ENTRYPOINT ["tini", "-s", "-g", "--", "/usr/bin/virt-prerunner"]

COPY build/virt-prerunner/iptables-wrapper /sbin/iptables-wrapper
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-wrapper 100

ADD build/virt-prerunner/virt-init-volume.sh /usr/bin/virt-init-volume
