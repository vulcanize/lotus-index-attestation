BIN = $(GOPATH)/bin

## Mockgen tool
MOCKGEN = $(BIN)/mockgen
$(BIN)/mockgen:
	go install github.com/golang/mock/mockgen@v1.6.0

MOCKS_DIR = $(CURDIR)/mocks

.PHONY: mocks test

mocks: $(MOCKGEN) mocks/types.go

mocks/snapshot/publisher.go: pkg/types/interfaces.go
	$(MOCKGEN) -package types_mock -destination $@ -source $< API AttestationService Checksummer ChecksumRepository

clean:
	rm -f mocks/types.go

build:
	go fmt ./...
	go build

test: mocks
	go clean -testcache && go test -p 1 -v ./...

dbtest: mocks
	go clean -testcache && TEST_WITH_DB=true go test -p 1 -v ./...
