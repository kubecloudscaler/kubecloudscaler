FROM cgr.dev/chainguard/static@sha256:1ff7590cbc50eaaa917c34b092de0720d307f67d6d795e4f749a0b80a2e95a2c
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY kubecloudscaler /
