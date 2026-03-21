package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"slices"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const timeout = 100 * time.Millisecond

func main() {
	log.SetOutput(os.Stderr)

	n := maelstrom.NewNode()
	kv := maelstrom.NewSeqKV(n)

	n.Handle("add", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		delta := body["delta"].(float64)

		for {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			old, err := kv.ReadInt(ctx, "g-counter")
			cancel()
			if err != nil {
				if rerr, ok := errors.AsType[*maelstrom.RPCError](err); ok && rerr.Code != maelstrom.KeyDoesNotExist {
					return err
				}
			}

			new := old + int(delta)

			ctx, cancel = context.WithTimeout(context.Background(), timeout)
			err = kv.CompareAndSwap(context.Background(), "g-counter", old, new, true)
			cancel()
			if err != nil {
				if rerr, ok := errors.AsType[*maelstrom.RPCError](err); ok && rerr.Code == maelstrom.PreconditionFailed {
					continue
				}
				return err
			}
			break
		}
		return n.Reply(msg, map[string]any{"type": "add_ok"})
	})
	n.Handle("read", func(msg maelstrom.Message) error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		v, err := kv.ReadInt(ctx, "g-counter")
		if err != nil {
			if rerr, ok := errors.AsType[*maelstrom.RPCError](err); ok && rerr.Code != maelstrom.KeyDoesNotExist {
				return err
			}
		}

		// It's inter-service read request, return our view of the counter.
		if slices.Contains(n.NodeIDs(), msg.Src) {
			return n.Reply(msg, map[string]any{"type": "read_ok", "value": v})
		}

		for _, dst := range n.NodeIDs() {
			if dst == n.ID() {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			m, err := n.SyncRPC(ctx, dst, map[string]any{"type": "read"})
			cancel()
			if err != nil {
				log.Println("failed to rpc with", dst)
				continue
			}
			var body map[string]any
			if err := json.Unmarshal(m.Body, &body); err != nil {
				return err
			}
			v = max(v, int(body["value"].(float64)))
		}
		return n.Reply(msg, map[string]any{"type": "read_ok", "value": v})
	})
	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
