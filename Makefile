.PHONY: prepare lint test vendor

export GO111MODULE=on

default: prepare lint test

prepare:
	tar -zxvf geolite2.tgz

lint:
	golangci-lint run

test:
	go test -v -cover ./...

vendor:
	go mod vendor

yaegi:
	yaegi run middleware.go

clean:
	rm -rf ./vendor *.mmdb || exit 0
