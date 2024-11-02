FROM cgr.dev/chainguard/static
LABEL org.opencontainers.image.source https://github.com/cloudscalerio/cloudscaler
ENTRYPOINT ["/cloudscaler"]
COPY cloudscaler /
