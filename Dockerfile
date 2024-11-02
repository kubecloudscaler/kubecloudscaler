FROM cgr.dev/chainguard/static@sha256:1c785f2145250a80d2d71d2b026276f3358ef3543448500c72206d37ec4ece37
LABEL org.opencontainers.image.source https://github.com/cloudscalerio/cloudscaler
ENTRYPOINT ["/cloudscaler"]
COPY cloudscaler /
