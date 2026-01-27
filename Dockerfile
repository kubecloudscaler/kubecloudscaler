FROM cgr.dev/chainguard/static@sha256:99a5f826e71115aef9f63368120a6aa518323e052297718e9bf084fb84def93c
ARG TARGETPLATFORM
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY $TARGETPLATFORM/kubecloudscaler /
