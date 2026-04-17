FROM cgr.dev/chainguard/static@sha256:6d508f497fe786ba47d57f4a3cffce12ca05c04e94712ab0356b94a93c4b457f
ARG TARGETPLATFORM
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY $TARGETPLATFORM/kubecloudscaler /
