
all:
	@GOPATH=`pwd` go install -v ./...

test:
	@GOPATH=`pwd` go test ./...

fmt:
	@GOPATH=`pwd` go fmt ./...

clean:
	@rm -rf bin pkg
