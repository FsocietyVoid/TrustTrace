# Makefile
build-prober:
	go build -o bin/prober ./cmd/prober

build-consensus:
	go build -o bin/consensus ./cmd/consensus

build-notary:
	go build -o bin/notary ./cmd/notary

run-prober: build-prober
	./bin/prober

clean:
	rm -rf bin/
