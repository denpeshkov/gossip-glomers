GOOS=linux GOARCH=amd64 go build -o bin/broadcast-3d ./cmd/broadcast-3d && \
docker run --rm \
	-v '.:/app' \
	maelstrom \
	maelstrom test -w broadcast --bin ./bin/broadcast-3d --node-count 25 --time-limit 20 --rate 100 --latency 100
