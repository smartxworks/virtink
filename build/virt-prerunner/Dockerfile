FROM golang:1.19-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN --mount=type=cache,target=/root/.cache/go-build go build -a cmd/virt-prerunner/main.go
RUN --mount=type=cache,target=/root/.cache/go-build go build -o rad -a cmd/route-advertisement-daemon/main.go

FROM alpine:3.17

RUN apk add --no-cache curl screen dnsmasq kea-dhcp6 cdrkit iptables ip6tables iproute2 qemu-virtiofsd dpkg util-linux s6-overlay nmap-ncat

RUN set -eux; \
    mkdir /var/lib/cloud-hypervisor; \
    case "$(uname -m)" in \
        'x86_64') \
            curl -sLo /usr/bin/cloud-hypervisor https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v28.0/cloud-hypervisor-static; \
            curl -sLo /usr/bin/ch-remote https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v28.0/ch-remote-static; \
            curl -sLo /var/lib/cloud-hypervisor/hypervisor-fw https://github.com/cloud-hypervisor/rust-hypervisor-firmware/releases/download/0.4.0/hypervisor-fw; \
            ;; \
        'aarch64') \
            curl -sLo /usr/bin/cloud-hypervisor https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v28.0/cloud-hypervisor-static-aarch64; \
            curl -sLo /usr/bin/ch-remote https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v28.0/ch-remote-static-aarch64; \
            curl -sLo /var/lib/cloud-hypervisor/CLOUDHV_EFI.fd https://github.com/smartxworks/cloud-hypervisor-edk2-builder/releases/download/20220706/CLOUDHV_EFI.fd; \
            ;; \
        *) echo >&2 "error: unsupported architecture '$(uname -m)'"; exit 1 ;; \
    esac; \
    chmod +x /usr/bin/cloud-hypervisor; \
    chmod +x /usr/bin/ch-remote

COPY build/virt-prerunner/cloud-hypervisor-type /etc/s6-overlay/s6-rc.d/cloud-hypervisor/type
COPY build/virt-prerunner/cloud-hypervisor-run.sh /etc/s6-overlay/s6-rc.d/cloud-hypervisor/run
COPY build/virt-prerunner/cloud-hypervisor-finish.sh /etc/s6-overlay/s6-rc.d/cloud-hypervisor/finish
RUN touch /etc/s6-overlay/s6-rc.d/user/contents.d/cloud-hypervisor

COPY --from=builder /workspace/main /usr/bin/virt-prerunner
COPY --from=builder /workspace/rad /usr/bin/rad
COPY build/virt-prerunner/virt-prerunner-type /etc/s6-overlay/s6-rc.d/virt-prerunner/type
COPY build/virt-prerunner/virt-prerunner-up /etc/s6-overlay/s6-rc.d/virt-prerunner/up
COPY build/virt-prerunner/virt-prerunner-run.sh /etc/s6-overlay/scripts/virt-prerunner-run.sh
RUN touch /etc/s6-overlay/s6-rc.d/user/contents.d/virt-prerunner
ENV S6_BEHAVIOUR_IF_STAGE2_FAILS=2

ENTRYPOINT ["/init"]

COPY build/virt-prerunner/iptables-wrapper /sbin/iptables-wrapper
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-wrapper 100
COPY build/virt-prerunner/ip6tables-wrapper /sbin/ip6tables-wrapper
RUN update-alternatives --install /sbin/ip6tables ip6tables /sbin/ip6tables-wrapper 100

ADD build/virt-prerunner/virt-init-volume.sh /usr/bin/virt-init-volume
