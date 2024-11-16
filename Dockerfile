FROM cgr.dev/chainguard/static@sha256:561b669256bd2b5a8afed34614e8cb1b98e4e2f66d42ac7a8d80d317d8c8688a
LABEL org.opencontainers.image.source https://github.com/cloudscalerio/cloudscaler
ENTRYPOINT ["/cloudscaler"]
COPY cloudscaler /
