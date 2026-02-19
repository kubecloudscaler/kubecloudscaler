FROM cgr.dev/chainguard/static@sha256:11ec91f0372630a2ca3764cea6325bebb0189a514084463cbb3724e5bb350d14
ARG TARGETPLATFORM
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY $TARGETPLATFORM/kubecloudscaler /
