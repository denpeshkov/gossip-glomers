GOOS=linux GOARCH=amd64 go build -o bin/kafka-log-5a ./cmd/kafka-log-5a && \
docker run --rm \
	-v '.:/app' \
	maelstrom \
	maelstrom test -w kafka --bin ./bin/kafka-log-5a --node-count 1 --concurrency 2n --time-limit 20 --rate 1000
