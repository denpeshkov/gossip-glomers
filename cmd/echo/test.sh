GOOS=linux GOARCH=amd64 go build -o bin/echo ./cmd/echo && \
docker run --rm \
	-v '.:/app' \
	maelstrom \
	maelstrom test -w echo --bin ./bin/echo --node-count 1 --time-limit 10
