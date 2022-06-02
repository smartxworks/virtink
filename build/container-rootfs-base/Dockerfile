FROM alpine

RUN apk add --no-cache e2fsprogs

ADD build/container-rootfs-base/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
