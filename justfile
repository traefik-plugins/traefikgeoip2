@_prepare:
	tar -xvzf geolite2.tgz

lint:
	golangci-lint run -v

test-go:
	go test -v -cover ./...

@_clean-yaegi:
  rm -rf /tmp/yaegi*

test-yaegi: && _clean-yaegi
  #!/bin/bash
  TMP=$(mktemp -d yaegi.XXXXXX -p /tmp)
  WRK="${TMP}/go/src/github.com/GiGInnovationLabs"
  mkdir -p ${WRK}
  ln -s `pwd` "${WRK}"
  cd "${WRK}/$(basename `pwd`)"
  env GOPATH="${TMP}/go" yaegi test -v .

test: _prepare lint test-go test-yaegi

clean:
  rm -rf *.mmdb
