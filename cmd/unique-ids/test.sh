GOOS=linux GOARCH=amd64 go build -o bin/unique-ids ./cmd/unique-ids && \
docker run --rm \
	-v '.:/app' \
	maelstrom \
	maelstrom test -w unique-ids --bin ./bin/unique-ids --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition
