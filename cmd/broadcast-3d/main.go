package main

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"os"
	"slices"
	"sync"
	"time"

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
	n    *maelstrom.Node
	mu   sync.RWMutex
	msgs map[float64]struct{}
}

func newNode() *node {
	n := &node{
		n:    maelstrom.NewNode(),
		msgs: make(map[float64]struct{}),
	}
	n.n.Handle("topology", n.handleTopology)
	n.n.Handle("broadcast", n.handleBroadcast)
	n.n.Handle("read", n.handleRead)
	n.n.Handle("gossip", n.handleGossip)
	n.n.Handle("broadcast_ok", func(msg maelstrom.Message) error { return nil })
	return n
}

func (n *node) run() error {
	return n.n.Run()
}

func (n *node) handleTopology(msg maelstrom.Message) error {
	// We are using full-mesh topology.
	return n.n.Reply(msg, map[string]any{"type": "topology_ok"})
}

func (n *node) handleBroadcast(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}
	id := body["message"].(float64)

	n.mu.Lock()
	defer n.mu.Unlock()
	if _, seen := n.msgs[id]; !seen {
		n.msgs[id] = struct{}{}
		for _, dst := range n.n.NodeIDs() {
			if dst == n.n.ID() {
				continue
			}
			go n.rpcWithRetry(dst, id)
		}
	}
	return n.n.Reply(msg, map[string]any{"type": "broadcast_ok"})
}

func (n *node) handleGossip(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}
	id := body["message"].(float64)

	n.mu.Lock()
	defer n.mu.Unlock()
	n.msgs[id] = struct{}{}
	return n.n.Reply(msg, map[string]any{"type": "gossip_ok"})
}

func (n *node) handleRead(msg maelstrom.Message) error {
	n.mu.RLock()
	defer n.mu.RUnlock()
	msgs := slices.Collect(maps.Keys(n.msgs))
	return n.n.Reply(msg, map[string]any{"type": "read_ok", "messages": msgs})
}

func (n *node) rpcWithRetry(dst string, id float64) {
	for {
		if err := n.rpc(dst, id); err == nil {
			return
		}
		log.Printf("failed to rpc with %s, retrying ...\n", dst)
		time.Sleep(200 * time.Millisecond)
	}
}

func (n *node) rpc(dst string, id float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := n.n.SyncRPC(ctx, dst, map[string]any{"type": "gossip", "message": id})
	return err
}
