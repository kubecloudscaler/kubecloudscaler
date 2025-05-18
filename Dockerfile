FROM cgr.dev/chainguard/static@sha256:8d1a96321dca0e8e7848b7db2d431191f15e7e302faa1428100bbab351d42c7a
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY kubecloudscaler /
