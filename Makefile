chartVersion := 0.1.1
imageVersion := latest

init:
	cp pkg/config.default.yaml config.yaml

protocol:
	cd proto && ./gen.sh

build:
	docker build --target build --platform=linux/amd64 --tag localhost:5001/blobcache:$(imageVersion) .
	docker push localhost:5001/blobcache:$(imageVersion)

start:
	cd hack; okteto up --file okteto.yml

stop:
	cd hack; okteto down --file okteto.yml

build-chart:
	helm package --dependency-update deploy/charts/blobcache --version $(chartVersion)

publish-chart:
	helm push beam-blobcache-v2-chart-$(chartVersion).tgz oci://public.ecr.aws/n4e0e1y0
	rm beam-blobcache-v2-chart-$(chartVersion).tgz

testclient:
	go build -o bin/testclient e2e/testclient/main.go
