GOOS=linux GOARCH=amd64 go build -o bin/kafka-log-5b ./cmd/kafka-log-5b && \
docker run --rm \
	-v '.:/app' \
	maelstrom \
	maelstrom test -w kafka --bin ./bin/kafka-log-5b --node-count 2 --concurrency 2n --time-limit 10 --rate 1000
