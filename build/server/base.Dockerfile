FROM ubuntu:16.04
LABEL maintainer="zczyi@xxxxx.cn"

COPY build/sources-amd64.list /etc/apt/sources.list

RUN apt-get update -y
RUN apt-get install -y tzdata wget ca-certificates python3 python3-pip
RUN wget http://mirrors.xxxxx.cn/tools/apache_hadoop/krb5.conf -O /etc/krb5.conf
RUN apt-get install -y libkrb5-dev krb5-user
RUN rm -f /etc/localtime \
    && ln -sv /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

