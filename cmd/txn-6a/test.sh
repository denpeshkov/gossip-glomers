GOOS=linux GOARCH=amd64 go build -o bin/txn-6a ./cmd/txn-6a && \
docker run --rm \
	-v '.:/app' \
	maelstrom \
	maelstrom test -w txn-rw-register --bin ./bin/txn-6a --node-count 1 --time-limit 20 --rate 1000 --concurrency 2n --consistency-models read-uncommitted --availability total
