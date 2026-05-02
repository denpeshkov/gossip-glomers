GOOS=linux GOARCH=amd64 go build -o bin/txn-datomic ./cmd/txn-datomic && \
docker run --rm \
	-v '.:/app' \
	maelstrom \
	maelstrom test -w txn-rw-register --bin ./bin/txn-datomic --node-count 3 --concurrency 2n --time-limit 20 --rate 1000 –-nemesis partition
