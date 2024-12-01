FROM cgr.dev/chainguard/static@sha256:5ff428f8a48241b93a4174dbbc135a4ffb2381a9e10bdbbc5b9db145645886d5
LABEL org.opencontainers.image.source https://github.com/k8scloudscaler/k8scloudscaler
ENTRYPOINT ["/k8scloudscaler"]
COPY k8scloudscaler /
