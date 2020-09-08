build:
	go build ./cmd/fruktsam

lint:
	golangci-lint run

deploy-dev: build
	curl -o reversecache "https://raw.githubusercontent.com/fruktkartan/fruktsam/master/reversecache"
	./fruktsam
	rsync -aP --chmod=ugo=rX dist/ steglits:web/fruktsam/
	git restore reversecache
