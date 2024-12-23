FROM cgr.dev/chainguard/static@sha256:f5fe67ab41c65f55766d824a2d857a7f56c9058b8e077c43d4d809c467f28df8
LABEL org.opencontainers.image.source https://github.com/k8scloudscaler/k8scloudscaler
ENTRYPOINT ["/k8scloudscaler"]
COPY k8scloudscaler /
