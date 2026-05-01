package main

import (
	"encoding/json"
	"log"
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
	n  *maelstrom.Node
	mu sync.RWMutex
	m  map[int]int
}

func newNode() *node {
	mn := maelstrom.NewNode()
	n := &node{n: mn, m: make(map[int]int)}
	n.n.Handle("txn", n.handleTxn)
	return n
}

func (n *node) run() error {
	return n.n.Run()
}

func (n *node) handleTxn(m maelstrom.Message) error {
	var req struct {
		Txns []txn `json:"txn"`
	}
	if err := json.Unmarshal(m.Body, &req); err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	for i := range req.Txns {
		txn := &req.Txns[i]
		switch txn.Op {
		case "r":
			txn.Val = new(n.m[txn.Key])
		case "w":
			n.m[txn.Key] = *txn.Val
		}
	}

	return n.n.Reply(m, map[string]any{"type": "txn_ok", "txn": req.Txns})
}

type txn struct {
	Op  string
	Key int
	Val *int
}

func (t *txn) UnmarshalJSON(b []byte) error {
	var arr [3]any
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}
	t.Op = arr[0].(string)
	t.Key = int(arr[1].(float64))
	if arr[2] == nil {
		t.Val = nil
	} else {
		t.Val = new(int(arr[2].(float64)))
	}
	return nil
}

func (t txn) MarshalJSON() ([]byte, error) {
	arr := []any{t.Op, t.Key, t.Val}
	return json.Marshal(arr)
}
