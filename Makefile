lint:
	golangci-lint run

build:
	go build ./cmd/fruktsam

deploy: build
	./fruktsam
	rsync -aP --chmod=ugo=rX dist/ steglits:web/fruktsam/
