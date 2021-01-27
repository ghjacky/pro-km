FROM ubuntu:16.04
LABEL maintainer="zczyi@xxxxx.cn"

COPY build/sources-amd64.list /etc/apt/sources.list
RUN apt-get update -y && apt-get install -y tzdata
RUN rm -f /etc/localtime \
    && ln -sv /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

COPY --from=galaxy_build:agent_amd64 /go/src/code.xxxxx.cn/platform/galaxy/_output/bin/amd64/deploy-agent /usr/local/bin/deploy-agent
CMD ["deploy-agent"]

