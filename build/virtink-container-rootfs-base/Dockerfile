FROM alpine:3.21.0

RUN apk add --no-cache e2fsprogs

ADD build/virtink-container-rootfs-base/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
