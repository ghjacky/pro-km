FROM harbor.xxxxx.cn/platform/deploy-manager:base
LABEL maintainer="zczyi@xxxxx.cn"

COPY --from=galaxy_build:server_amd64  /go/src/code.xxxxx.cn/platform/galaxy/_output/bin/amd64/deploy-manager /usr/local/bin/deploy-manager
CMD ["deploy-manager"]
