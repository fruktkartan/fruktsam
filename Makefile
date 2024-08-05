
.PHONY: fruktsam
fruktsam:
	go build ./cmd/fruktsam

run: fruktsam
	curl -s -o reversecache "https://raw.githubusercontent.com/fruktkartan/fruktsam/master/reversecache"
	./fruktsam
	git restore reversecache

deploy-dev: run
	rsync -aP --chmod=ugo=rX dist/ lublin.se:/home/frukt/fruktsam/dev/

simple-run: fruktsam
	./fruktsam

simple-deploy-dev: simple-run
	rsync -aP --chmod=ugo=rX dist/ lublin.se:/home/frukt/fruktsam/dev/

golangci_version=v1.59.1
golangci_cachedir=$(HOME)/.cache/golangci-lint-$(golangci_version)
.PHONY: lint
lint:
	mkdir -p $(golangci_cachedir)
	podman run --rm -it \
		-v $$(pwd):/src -w /src \
		-v $(golangci_cachedir):/root/.cache \
		docker.io/golangci/golangci-lint:$(golangci_version)-alpine \
		golangci-lint run
