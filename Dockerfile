FROM cgr.dev/chainguard/static@sha256:3348c5f7b97a4d63944034a8c6c43ad8bc69771b2564bed32ea3173bc96b4e04
ARG TARGETPLATFORM
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY $TARGETPLATFORM/kubecloudscaler /
