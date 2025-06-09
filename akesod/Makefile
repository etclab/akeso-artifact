progs = aesgcm cloud-cp akesod gcs-utils trigger-key-update

all: $(progs)

$(progs): % : vet
	go build ./cmd/$@

vet: fmt
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -f $(progs)

clean_keys:
	rm -rf keys/*.pem
	rm -f keys/*.msg
	rm -f keys/*.msg.sig
	rm -f keys/*.json
	rm -f keys/*.msg.mac

.PHONY: all vet fmt clean
