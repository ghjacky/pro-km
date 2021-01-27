# Copyright 2020 xxxxx Inc.

REGISTRY=harbor.xxxxx.cn

PROJECT_PACKAGE=code.xxxxx.cn/platform/galaxy
GIT_COMMIT=$(shell git rev-parse "HEAD^{commit}")
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
git_status=$(shell git status --porcelain)
ifeq "${git_status}" ""
GIT_TREE_STATE=clean
else
GIT_TREE_STATE=dirty
endif

VERSION=$(shell git describe --tags --abbrev=7 HEAD --always)
GIT_MAJOR=$(word 1,$(subst ., ,$(VERSION:v%=%)))
GIT_MINOR=$(word 2,$(subst ., ,$(VERSION:v%=%)))
LDFLAGS_X=-X "${PROJECT_PACKAGE}/pkg/version.gitCommit=${GIT_COMMIT}" \
		  -X "${PROJECT_PACKAGE}/pkg/version.gitTreeState=${GIT_TREE_STATE}" \
		  -X "${PROJECT_PACKAGE}/pkg/version.gitVersion=${VERSION}" \
		  -X "${PROJECT_PACKAGE}/pkg/version.gitMajor=${GIT_MAJOR}" \
		  -X "${PROJECT_PACKAGE}/pkg/version.gitMinor=${GIT_MINOR}" \
	   	  -X "${PROJECT_PACKAGE}/pkg/version.buildDate=${BUILD_DATE}"

IMAGE_DEPLOY_MANAGER_NAME=${REGISTRY}/platform/deploy-manager:${VERSION}
IMAGE_DEPLOY_AGENT_NAME=${REGISTRY}/platform/deploy-agent:${VERSION}
TEST_MANAGER_ADDR="172.16.244.6:32270"
TEST_MANAGER_PROVIDER_ADDR="http://172.16.244.6:32280/api/v1/provider"
LOCAL_PMP_SECRET="enlbcrKAZB1VHvxkINik"

PROJECT_PATH=src/${PROJECT_PACKAGE}
LOCAL_PROJECT_PATH=$(PWD)
ifeq "$(GITLAB_CI)" "true"
LOCAL_PROJECT_PATH=$(shell docker ps | grep ${HOSTNAME} | awk '{print $$1}' | xargs docker inspect --format "{{range .Mounts}}{{.Destination}} {{.Source}}{{println}}{{end}}" | grep /builds | awk '{print $$2}')/platform/galaxy
endif
DEPLOY_APP=galaxy

RPOTO_FOLDER=pkg/component/cmd/v1
PROTO_SOURCE=$(shell find $(RPOTO_FOLDER) -name '*.proto')
.PHONY: proto_generate
proto_generate:
		echo find in folder ${RPOTO_FOLDER}
		for src in $(PROTO_SOURCE); do \
			echo generating $$src; \
			docker run --rm -v $(LOCAL_PROJECT_PATH):$(PWD) -w $(PWD) znly/protoc --go_out=plugins=grpc:. -I. $$src; \
		done

.PHONY: image_clean
image_clean:
		docker image prune -f

.PHONY: env
env:
		docker build \
			-t ${REGISTRY}/platform/galaxy_build:amd64 \
			-f build/env/Dockerfile .

.PHONY: push_env
push_env:
		docker login ${REGISTRY}
		docker push ${REGISTRY}/platform/galaxy_build:amd64

.PHONY: pull_env
pull_env:
		docker pull ${REGISTRY}/platform/galaxy_build:amd64

.PHONY: server_base
server_base:
		docker build \
			-t ${REGISTRY}/platform/deploy-manager:base \
			-f build/server/base.Dockerfile .

.PHONY: build_server
build_server:
		docker build --force-rm -f build/env/build.Dockerfile \
		    -t galaxy_build:server_amd64 \
		    --build-arg LDFLAGS_X='${LDFLAGS_X}' \
		    --build-arg OUTPUT=_output/bin/amd64/deploy-manager \
		    --build-arg CMD_PATH=cmd/server/server.go \
		    --squash --compress \
		    .

.PHONY: build_agent
build_agent:
		docker build --force-rm -f build/env/build.Dockerfile \
        	-t galaxy_build:agent_amd64 \
        	--build-arg LDFLAGS_X='${LDFLAGS_X}' \
        	--build-arg OUTPUT=_output/bin/amd64/deploy-agent \
        	--build-arg CMD_PATH=cmd/agent/agent.go \
        	--squash --compress \
        	.

.PHONY: build
build: pull_env verify proto_generate build_server build_agent image_clean

.PHONY: server
server: pull_env verify proto_generate build_server
		docker build \
			--rm . \
			-t ${IMAGE_DEPLOY_MANAGER_NAME} \
			-f build/server/Dockerfile

.PHONY: run_server
run_server:
		go run cmd/server/server.go \
            --key=deploy/cert/manager/tls.key \
            --cert=deploy/cert/manager/tls.crt \
            --ca=deploy/cert/manager/ca.crt \
            --db=deploy/db/idc/config.yaml \
            --auth-addr=http://172.16.244.6:30099 \
            --auth-client-id=68 \
            --auth-client-secret=bl-hezU8XyUw4uu-ut4ezw \
            --auth-skip=true \
            --web-insecure=true \
            --v=10\
            --swagger-enable=true\
            --storage-dir=${LOCAL_PROJECT_PATH}/.data\
            --pmp-secret=enlbcrKAZB1VHvxkINik

.PHONY: run_agent
run_agent:
		mkdir -p ${LOCAL_PROJECT_PATH}/.run/data/agent
		go run cmd/agent/agent.go \
			--id=local-${USER} \
			--config=${LOCAL_PROJECT_PATH}/deploy/agent/test/idc/config.yaml \
			--work-dir=${LOCAL_PROJECT_PATH}/.run/data/agent \
			--manager-addr=${TEST_MANAGER_ADDR} \
			--manager-provider-addr=${TEST_MANAGER_PROVIDER_ADDR} \
			--manager-cert=${LOCAL_PROJECT_PATH}/deploy/cert/agent/agent.p12 \
			--pmp-secret=${LOCAL_PMP_SECRET}
			--ttl=1m \
			--v=10

.PHONY: agent
agent: pull_env verify build_agent
		docker build \
			--rm . \
			-t ${IMAGE_DEPLOY_AGENT_NAME} \
			-f build/agent/Dockerfile


.PHONY: push_server
push_server:
		docker login ${REGISTRY}
		docker push ${IMAGE_DEPLOY_MANAGER_NAME}

.PHONY: push_agent
push_agent:
		docker login ${REGISTRY}
		docker push ${IMAGE_DEPLOY_AGENT_NAME}

.PHONY: push
push:
		docker login ${REGISTRY}
		docker push ${IMAGE_DEPLOY_MANAGER_NAME}
		docker push ${IMAGE_DEPLOY_AGENT_NAME}

.PHONY: vendor
vendor:
		docker run --rm -e GO111MODULE=on -v $(PWD):/go/${PROJECT_PATH} -v ${GOPATH}/pkg/:/go/pkg -w /go/${PROJECT_PATH} golang:1.13 bash -c "hack/update-vendor.sh"


.PHONY: deploy
deploy:
		docker run --rm \
			--add-host bsy-cdagent.xxxxx.cn:49.234.243.43 \
			--add-host bdy-cdagent.xxxxx.cn:106.12.80.151 \
			harbor.xxxxx.cn/platform/atlas_deploy_client:v0.4 ./client --app ${DEPLOY_APP} --host ${DEPLOY_HOST} --secret ${DEPLOY_SECRET} --tag ${VERSION}

.PHONY: verify
verify:
		docker run --rm \
			-v $(LOCAL_PROJECT_PATH):/go/${PROJECT_PATH} \
			-w /go/${PROJECT_PATH} \
			${REGISTRY}/platform/galaxy_build:amd64 \
			hack/verify-all.sh
