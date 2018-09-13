.DEFAULT_GOAL := all
REPO_URL=harbor.nuglif.net/admin
APP=versions_exporter

.PHONY: build-image clean

build: ## Build the docker image
	docker build . -t ${REPO_URL}/${APP}:${TAG} --no-cache
#	docker tag versions_exporter:${TAG} versions_exporter:latest

push:
	docker push ${REPO_URL}/${APP}:${TAG}

clean:
	docker image rm ${APP}:${TAG}

all: build push