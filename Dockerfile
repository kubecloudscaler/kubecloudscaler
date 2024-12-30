FROM cgr.dev/chainguard/static@sha256:95a45fc5fda9aa71dbdc645b20c6fb03f33aec8c1c2581ef7362b1e6e1d09dfb
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY kubecloudscaler /
