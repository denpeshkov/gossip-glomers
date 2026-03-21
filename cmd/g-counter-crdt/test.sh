GOOS=linux GOARCH=amd64 go build -o bin/g-counter-crdt ./cmd/g-counter-crdt && \
docker run --rm \
	-v '.:/app' \
	maelstrom \
	maelstrom test -w g-counter --bin ./bin/g-counter-crdt --node-count 3 --rate 100 --time-limit 20 --nemesis partition
