GOOS=linux GOARCH=amd64 go build -o bin/kafka-log-5c ./cmd/kafka-log-5c && \
docker run --rm \
	-v '.:/app' \
	maelstrom \
	maelstrom test -w kafka --bin ./bin/kafka-log-5c --node-count 2 --concurrency 2n --time-limit 20 --rate 1000
