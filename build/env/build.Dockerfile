FROM harbor.xxxxx.cn/platform/galaxy_build:amd64

WORKDIR /go/src/code.xxxxx.cn/platform/galaxy

ENV CGO_ENABLED=1

ARG LDFLAGS_X
ARG OUTPUT
ARG CMD_PATH

COPY . .

RUN echo "$LDFLAGS_X" && \
    echo ${OUTPUT} && \
    echo ${CMD_PATH} && \
    go build -v -mod=vendor -ldflags "${LDFLAGS_X}" -tags "kerberos" -o ${OUTPUT} ${CMD_PATH}
