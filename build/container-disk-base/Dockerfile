FROM alpine

RUN apk add --no-cache qemu-img

ADD build/container-disk-base/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
