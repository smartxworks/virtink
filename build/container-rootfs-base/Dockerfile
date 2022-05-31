FROM alpine

RUN apk add e2fsprogs

ADD build/container-rootfs-base/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
