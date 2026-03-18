package main

import (
	"log"

	"github.com/google/uuid"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	n.Handle("generate", func(msg maelstrom.Message) error {
		body := struct {
			Type string    `json:"type"`
			ID   uuid.UUID `json:"id"`
		}{
			Type: "generate_ok",
			ID:   uuid.New(),
		}
		return n.Reply(msg, body)
	})
	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
