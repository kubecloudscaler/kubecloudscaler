FROM cgr.dev/chainguard/static@sha256:f96b5a60658dfee0cae426972afecad6ea6930fa28e6d8ef7096a7bdf35d6498
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY kubecloudscaler /
