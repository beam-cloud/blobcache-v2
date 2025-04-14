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
	cd hack && okteto up --file okteto.yaml

stop:
	cd hack && okteto down --file okteto.yaml

build-chart:
	helm package --dependency-update deploy/charts/blobcache --version $(chartVersion)

publish-chart:
	helm push beam-blobcache-v2-chart-$(chartVersion).tgz oci://public.ecr.aws/n4e0e1y0
	rm beam-blobcache-v2-chart-$(chartVersion).tgz

testclients:
	GOOS=darwin GOARCH=arm64 go build -o bin/throughput e2e/throughput/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/fs e2e/fs/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/basic e2e/basic/main.go

setup: build
	@if [ "$(shell kubectl config current-context)" != "k3d-beta9" ]; then \
		echo "Current context is not k3d-beta9"; \
		exit 1; \
	fi
	helm install blobcache-valkey oci://registry-1.docker.io/bitnamicharts/valkey --set architecture=standalone --set auth.password=password
	cd hack; kubectl apply -f deployment.yaml; cd ..