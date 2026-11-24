FROM cgr.dev/chainguard/static@sha256:599c3cd7597b84aa149c4282beecc8c6d4e1cb5c95dc8d982e70cd0a2a9147a2
ARG TARGETPLATFORM
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY $TARGETPLATFORM/kubecloudscaler /
