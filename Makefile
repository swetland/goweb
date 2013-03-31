
all:
	@GOPATH=`pwd` go install -v ./...

test:
	@GOPATH=`pwd` go test ./...

clean:
	@rm -rf bin pkg
