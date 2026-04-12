package main

import (
	"encoding/json"
	"log"
	"maps"
	"os"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	log.SetOutput(os.Stderr)

	n := newNode()
	if err := n.run(); err != nil {
		log.Fatal(err)
	}
}

type node struct {
	n               *maelstrom.Node
	mu              sync.RWMutex
	logs            map[string][]int
	commitedOffsets map[string]int
}

func newNode() *node {
	n := &node{
		n:               maelstrom.NewNode(),
		commitedOffsets: make(map[string]int),
		logs:            make(map[string][]int),
	}
	n.n.Handle("send", n.handleSend)
	n.n.Handle("poll", n.handlePoll)
	n.n.Handle("commit_offsets", n.commitOffsets)
	n.n.Handle("list_committed_offsets", n.listCommitedOffsets)
	return n
}

func (n *node) run() error {
	return n.n.Run()
}

func (n *node) handleSend(m maelstrom.Message) error {
	var req struct {
		Key string `json:"key"`
		Msg int    `json:"msg"`
	}
	if err := json.Unmarshal(m.Body, &req); err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	off := len(n.logs[req.Key])
	n.logs[req.Key] = append(n.logs[req.Key], req.Msg)

	return n.n.Reply(m, map[string]any{"type": "send_ok", "offset": off})
}

func (n *node) handlePoll(m maelstrom.Message) error {
	var req struct {
		Offsets map[string]int `json:"offsets"`
	}
	if err := json.Unmarshal(m.Body, &req); err != nil {
		return err
	}

	n.mu.RLock()
	defer n.mu.RUnlock()
	msgs := make(map[string][][2]int, len(req.Offsets))
	for key, off := range req.Offsets {
		lg := n.logs[key][off:]

		entries := make([][2]int, len(lg))
		for i, v := range lg {
			entries[i] = [2]int{off + i, v}
		}
		msgs[key] = entries
	}

	resp := struct {
		Type string              `json:"type"`
		Msgs map[string][][2]int `json:"msgs"`
	}{
		Type: "poll_ok",
		Msgs: msgs,
	}
	return n.n.Reply(m, resp)
}

func (n *node) commitOffsets(m maelstrom.Message) error {
	var req struct {
		Offsets map[string]int `json:"offsets"`
	}
	if err := json.Unmarshal(m.Body, &req); err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	maps.Copy(n.commitedOffsets, req.Offsets)

	return n.n.Reply(m, map[string]any{"type": "commit_offsets_ok"})
}

func (n *node) listCommitedOffsets(m maelstrom.Message) error {
	var req struct {
		Keys []string `json:"keys"`
	}
	if err := json.Unmarshal(m.Body, &req); err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	offsets := make(map[string]int, len(req.Keys))
	for _, key := range req.Keys {
		offsets[key] = n.commitedOffsets[key]
	}

	resp := struct {
		Type    string         `json:"type"`
		Offsets map[string]int `json:"offsets"`
	}{
		Type:    "list_committed_offsets_ok",
		Offsets: offsets,
	}
	return n.n.Reply(m, resp)
}
