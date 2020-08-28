lint:
	golangci-lint run

build:
	go build

deploy: build
	rsync -aP --chmod=ugo=rX dist/ steglits:web/fruktsam/
