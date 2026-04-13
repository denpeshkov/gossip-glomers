package main

import (
	"context"
	"encoding/json"
	"errors"
	"hash/crc32"
	"log"
	"os"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const numPartitions = 4
const (
	logPrefix             = "log_"
	commitedOffsetsPrefix = "commited_offsets_"
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
	kv *maelstrom.KV
	mu sync.RWMutex
}

func newNode() *node {
	mn := maelstrom.NewNode()
	kv := maelstrom.NewLinKV(mn)
	n := &node{n: mn, kv: kv}
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

	nodeID := int(crc32.ChecksumIEEE([]byte(req.Key))) % len(n.n.NodeIDs())
	if dest := n.n.NodeIDs()[nodeID]; dest != n.n.ID() {
		resp, err := n.n.SyncRPC(context.Background(), dest, m.Body)
		if err != nil {
			return err
		}

		var off struct {
			Offset int `json:"offset"`
		}
		if err := json.Unmarshal(resp.Body, &off); err != nil {
			return err
		}
		return n.n.Reply(m, map[string]any{"type": "send_ok", "offset": off.Offset})
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	var lg []int
	if err := n.kv.ReadInto(context.Background(), logPrefix+req.Key, &lg); err != nil {
		if rerr, ok := errors.AsType[*maelstrom.RPCError](err); !ok || rerr.Code != maelstrom.KeyDoesNotExist {
			return err
		}
	}

	lg = append(lg, req.Msg)
	if err := n.kv.Write(context.Background(), logPrefix+req.Key, lg); err != nil {
		if rerr, ok := errors.AsType[*maelstrom.RPCError](err); !ok || rerr.Code != maelstrom.PreconditionFailed {
			return err
		}
	}

	return n.n.Reply(m, map[string]any{"type": "send_ok", "offset": len(lg) - 1})
}

func (n *node) handlePoll(m maelstrom.Message) error {
	var req struct {
		Offsets map[string]int `json:"offsets"`
	}
	if err := json.Unmarshal(m.Body, &req); err != nil {
		return err
	}

	msgs := make(map[string][][2]int, len(req.Offsets))
	for key, off := range req.Offsets {
		var lg []int
		if err := n.kv.ReadInto(context.Background(), logPrefix+key, &lg); err != nil {
			if rerr, ok := errors.AsType[*maelstrom.RPCError](err); !ok || rerr.Code != maelstrom.KeyDoesNotExist {
				return err
			}
		}
		lg = lg[off:]

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

	for key, off := range req.Offsets {
		if err := n.kv.Write(context.Background(), commitedOffsetsPrefix+key, off); err != nil {
			if rerr, ok := errors.AsType[*maelstrom.RPCError](err); !ok || rerr.Code != maelstrom.PreconditionFailed {
				return err
			}
		}
	}

	return n.n.Reply(m, map[string]any{"type": "commit_offsets_ok"})
}

func (n *node) listCommitedOffsets(m maelstrom.Message) error {
	var req struct {
		Keys []string `json:"keys"`
	}
	if err := json.Unmarshal(m.Body, &req); err != nil {
		return err
	}

	offsets := make(map[string]int, len(req.Keys))
	for _, key := range req.Keys {
		off, err := n.kv.ReadInt(context.Background(), commitedOffsetsPrefix+key)
		if err != nil {
			if err, ok := errors.AsType[*maelstrom.RPCError](err); ok && err.Code == maelstrom.KeyDoesNotExist {
				continue
			}
			return err
		}
		offsets[key] = off
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
