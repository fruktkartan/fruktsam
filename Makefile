build:
	go build ./cmd/fruktsam

lint:
	golangci-lint run

deploy: build
	./fruktsam
	rsync -aP --chmod=ugo=rX dist/ steglits:web/fruktsam/
