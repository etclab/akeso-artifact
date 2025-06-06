progs= alice bob key-manager

all:  $(progs)

$(progs): % : vet
	go build ./cmd/$@

vet: fmt
	go vet ./...

fmt:
	go fmt ./...

drat-bench:
	go test ./internal/key-manager -v -bench=. -benchmem

clean:
	rm -f $(progs)

.PHONY: all vet fmt clean
