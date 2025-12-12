FROM cgr.dev/chainguard/static@sha256:a301031ffd4ed67f35ca7fa6cf3dad9937b5fa47d7493955a18d9b4ca5412d1a
ARG TARGETPLATFORM
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY $TARGETPLATFORM/kubecloudscaler /
