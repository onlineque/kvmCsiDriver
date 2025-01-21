FROM golang:1.23.5-alpine AS build-stage

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && \
    apk add upx binutils tzdata

COPY *.go ./
COPY csi_proto/*.go ./csi_proto/
COPY storageagent_proto/*.go ./storageagent_proto/
COPY cmd/*.go ./cmd/
COPY pkg/driver/*.go ./pkg/driver/
COPY pkg/kvm/*.go ./pkg/kvm/

RUN CGO_ENABLED=0 GOOS=linux go build -o /driver && \
    strip /driver && \
    upx --ultra-brute /driver

# FROM scratch AS final
FROM golang:1.23.5-bookworm AS final
LABEL org.opencontainers.image.description="PoC testing Kubernetes CSI driver"
LABEL org.opencontainers.image.authors="Vladimir Siman (https://github.com/onlineque)"
LABEL org.opencontainers.image.source="https://github.com/onlineque/kvmCsiDriver"
WORKDIR /
COPY --from=build-stage /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build-stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-stage /driver /driver
# USER 10001:10001
ENTRYPOINT ["/driver"]
