build:
	go build ./cmd/fruktsam

run: build
	curl -s -o reversecache "https://raw.githubusercontent.com/fruktkartan/fruktsam/master/reversecache"
	./fruktsam
	git restore reversecache

lint:
	golangci-lint run

deploy-dev: run
	rsync -aP --chmod=ugo=rX dist/ steglits:web/fruktsam/
