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
	cd hack; okteto up --file okteto.yaml

stop:
	cd hack; okteto down --file okteto.yaml

build-chart:
	helm package --dependency-update deploy/charts/blobcache --version $(chartVersion)

publish-chart:
	helm push beam-blobcache-v2-chart-$(chartVersion).tgz oci://public.ecr.aws/n4e0e1y0
	rm beam-blobcache-v2-chart-$(chartVersion).tgz

testclients:
	GOOS=linux GOARCH=amd64 go build -o bin/throughput e2e/throughput/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/fs e2e/fs/main.go
