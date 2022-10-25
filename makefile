SHELL := /bin/bash

# =================================================================
# Building containers

list-crawler:
	docker build \
		-f docker/Dockerfile.list-crawler \
		-t list-crawler-amd64:1.0 \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		.
# =================================================================
# Running from within k8s/dev

kind-up:
	kind create cluster --image kindest/node:v1.24.2 --name service-starter-cluster --config k8s/dev/kind-config.yaml

kind-down:
	kind delete cluster --name service-starter-cluster

kind-load:
	kind load docker-image list-crawler-amd64:1.0 --name service-starter-cluster

kind-services:
	kustomize build k8s/dev | kubectl apply -f -

kind-list-crawler: list-crawler
	kind load docker-image list-crawler-amd64:1.0 --name service-starter-cluster
	kubectl delete pods -lapp=list-crawler

kind-logs:
	kubectl logs -lapp=list-crawler --all-containers=true -f

kind-status:
	kubectl get nodes
	kubectl get pods --watch

kind-status-full:
	kubectl describe pod -lapp=list-crawler

# =================================================================

run:
	go run cmd/crawler/main.go

tidy:
	go mod tidy