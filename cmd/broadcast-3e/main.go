package main

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"math/rand/v2"
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
	n       *maelstrom.Node
	mu      sync.RWMutex
	msgs    map[float64]struct{}
	pending map[float64]struct{}
}

func newNode() *node {
	n := &node{
		n:       maelstrom.NewNode(),
		msgs:    make(map[float64]struct{}),
		pending: make(map[float64]struct{}),
	}
	n.n.Handle("topology", n.handleTopology)
	n.n.Handle("broadcast", n.handleBroadcast)
	n.n.Handle("read", n.handleRead)
	n.n.Handle("gossip", n.handleGossip)
	n.n.Handle("broadcast_ok", func(msg maelstrom.Message) error { return nil })
	return n
}

func (n *node) run() error {
	go n.gossip() // FIXME: Leaks.
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
		n.pending[id] = struct{}{}
	}
	return n.n.Reply(msg, map[string]any{"type": "broadcast_ok"})
}

func (n *node) handleGossip(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}
	ids := body["messages"].([]any)

	n.mu.Lock()
	defer n.mu.Unlock()
	for _, id := range ids {
		n.msgs[id.(float64)] = struct{}{}
	}
	return n.n.Reply(msg, map[string]any{"type": "gossip_ok"})
}

func (n *node) handleRead(msg maelstrom.Message) error {
	n.mu.RLock()
	defer n.mu.RUnlock()
	msgs := slices.Collect(maps.Keys(n.msgs))
	return n.n.Reply(msg, map[string]any{"type": "read_ok", "messages": msgs})
}

func (n *node) gossip() {
	t := time.NewTicker(1 * time.Second)
	for range t.C {
		n.mu.Lock()
		ids := slices.Collect(maps.Keys(n.pending))
		clear(n.pending)
		n.mu.Unlock()

		if len(ids) == 0 {
			continue
		}
		for _, dst := range n.n.NodeIDs() {
			if dst == n.n.ID() {
				continue
			}
			go n.rpcWithRetry(dst, ids)
		}
	}
}

func (n *node) rpcWithRetry(dst string, msgs []float64) {
	for {
		if err := n.rpc(dst, msgs); err == nil {
			return
		}
		log.Printf("failed to rpc with %s, retrying ...\n", dst)
		time.Sleep(200 * time.Millisecond)
	}
}

func (n *node) rpc(dst string, msgs []float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := n.n.SyncRPC(ctx, dst, map[string]any{"type": "gossip", "messages": msgs})
	return err
}

// sample returns a random sample of k neighbours.
func (n *node) sample(k int) []string {
	nodes := slices.Clone(n.n.NodeIDs())
	nodes = slices.DeleteFunc(nodes, func(id string) bool { return id == n.n.ID() })
	rand.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
	return nodes[:min(k, len(nodes))]
}
