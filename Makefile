.DEFAULT_GOAL := all
REPO_URL=harbor.nuglif.net/nuglif
APP=versions_exporter
TAG=0.1.7

.PHONY: build-image clean

build: ## Build the docker image
	docker build . -t ${REPO_URL}/${APP}:${TAG} --no-cache
#	docker tag versions_exporter:${TAG} versions_exporter:latest

push:
	docker push ${REPO_URL}/${APP}:${TAG}

clean:
	docker image rm ${APP}:${TAG}

all: build push
