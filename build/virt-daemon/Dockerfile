FROM golang:1.19-alpine AS builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -a cmd/virt-daemon/main.go

FROM alpine

COPY --from=builder /workspace/main /usr/bin/virt-daemon
ENTRYPOINT ["virt-daemon"]
