FROM cgr.dev/chainguard/static@sha256:7a6456cc96ecde793b7c8ad9a3ccd5d610d6168a6f64d693ecc2e84f8276c6c6
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY kubecloudscaler /
