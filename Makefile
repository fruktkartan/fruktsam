
.PHONY: fruktsam
fruktsam:
	go build ./cmd/fruktsam

run: fruktsam
	curl -s -o reversecache "https://raw.githubusercontent.com/fruktkartan/fruktsam/master/reversecache"
	./fruktsam
	git restore reversecache

.PHONY: lint
lint:
	make -C gotools golangci-lint
	./gotools/golangci-lint run

deploy-dev: run
	rsync -aP --chmod=ugo=rX dist/ lublin.se:/home/frukt/fruktsam/dev/

simple-run: fruktsam
	./fruktsam

simple-deploy-dev: simple-run
	rsync -aP --chmod=ugo=rX dist/ lublin.se:/home/frukt/fruktsam/dev/
