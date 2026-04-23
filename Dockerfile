FROM cgr.dev/chainguard/static@sha256:bee469c98ce2df388a2746dc360bd59eb3efe2dab366a01cdcbfd738a2ca1474
ARG TARGETPLATFORM
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY $TARGETPLATFORM/kubecloudscaler /
