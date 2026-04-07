.PHONY: test test-bootstrap

test:
	go test ./...

test-bootstrap:
	docker build -f test/docker/bootstrap.Dockerfile .
