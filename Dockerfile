FROM cgr.dev/chainguard/static@sha256:60582b2ae6074f641094af0f370d4ab241aab271858a66223dcde7eee9f51638
ARG TARGETPLATFORM
LABEL org.opencontainers.image.source https://github.com/kubecloudscaler/kubecloudscaler
ENTRYPOINT ["/kubecloudscaler"]
COPY $TARGETPLATFORM/kubecloudscaler /
