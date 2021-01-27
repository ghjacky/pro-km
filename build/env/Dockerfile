FROM golang:1.13

WORKDIR /go/src/code.xxxxx.cn/platform/galaxy

RUN apt-get update -y \
&& apt-get install -y libkrb5-dev \
&& go env -w GO111MODULE=on \
&& go env -w GOPROXY="https://goproxy.cn,direct" \
&& go get golang.org/x/tools/cmd/goimports@v0.0.0-20210112230658-8b4aab62c064 \
&& go get -u golang.org/x/lint/golint

COPY . .

RUN go mod tidy &&\
    go mod vendor &&\
    mv vendor ../vendor &&\
    rm -rf ./* &&\
    mv ../vendor ./
