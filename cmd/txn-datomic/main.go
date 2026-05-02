package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
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
	kv *maelstrom.KV
}

func newNode() *node {
	mn := maelstrom.NewNode()
	n := &node{
		n:  mn,
		kv: maelstrom.NewLinKV(mn),
	}
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

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var stateID string
		if err := n.kv.ReadInto(ctx, "root_pointer", &stateID); err != nil {
			if rerr, ok := errors.AsType[*maelstrom.RPCError](err); !ok || rerr.Code != maelstrom.KeyDoesNotExist {
				return err
			}
		}

		state := state{kv: n.kv, m: make(map[string]*thunk)}
		if stateID != "" {
			if err := n.kv.ReadInto(ctx, stateID, &state); err != nil {
				if rerr, ok := errors.AsType[*maelstrom.RPCError](err); !ok || rerr.Code != maelstrom.KeyDoesNotExist {
					return err
				}
			}
		}

		for i := range req.Txns {
			txn := &req.Txns[i]
			switch txn.Op {
			case "r":
				thu, ok := state.m[txn.Key]
				if !ok {
					txn.Val = nil
					break
				}
				val, err := thu.load(ctx)
				if err != nil {
					return err
				}
				txn.Val = val
			case "w":
				thu, ok := state.m[txn.Key]
				if ok && !thu.saved {
					thu.value = txn.Val
					break
				}
				thu = &thunk{kv: n.kv, id: uuid.NewString(), value: txn.Val, saved: false}
				state.m[txn.Key] = thu
			}
		}

		for _, thu := range state.m {
			if err := thu.store(ctx); err != nil {
				return err
			}
		}

		newStateID := uuid.NewString()
		if err := n.kv.Write(ctx, newStateID, state); err != nil {
			return err
		}

		if err := n.kv.CompareAndSwap(ctx, "root_pointer", stateID, newStateID, true); err != nil {
			if rerr, ok := errors.AsType[*maelstrom.RPCError](err); ok && rerr.Code == maelstrom.PreconditionFailed {
				continue
			}
			return err
		}
		break
	}

	return n.n.Reply(m, map[string]any{"type": "txn_ok", "txn": req.Txns})
}

type state struct {
	kv *maelstrom.KV
	m  map[string]*thunk // key -> thunk
}

func (s *state) UnmarshalJSON(b []byte) error {
	var m map[string]string // key -> thunk_id
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	s.m = make(map[string]*thunk, len(m))
	for key, thunkID := range m {
		s.m[key] = &thunk{kv: s.kv, id: thunkID, value: nil, saved: true}
	}
	return nil
}

func (s state) MarshalJSON() ([]byte, error) {
	m := make(map[string]string, len(s.m))
	for key, thunk := range s.m {
		m[key] = thunk.id
	}
	return json.Marshal(m)
}

type thunk struct {
	kv    *maelstrom.KV
	id    string
	value *int
	saved bool
}

func (t *thunk) load(ctx context.Context) (*int, error) {
	if t.value != nil {
		return t.value, nil
	}
	if !t.saved {
		return nil, nil
	}

	v, err := t.kv.ReadInt(ctx, t.id)
	if err != nil {
		if rerr, ok := errors.AsType[*maelstrom.RPCError](err); ok && rerr.Code == maelstrom.KeyDoesNotExist {
			return nil, nil
		}
		return nil, err
	}
	t.value = &v
	return t.value, nil
}

func (t *thunk) store(ctx context.Context) error {
	if t.saved || t.value == nil {
		return nil
	}
	if err := t.kv.Write(ctx, t.id, *t.value); err != nil {
		return err
	}
	t.saved = true
	return nil
}

type txn struct {
	Op  string
	Key string
	Val *int
}

func (t *txn) UnmarshalJSON(b []byte) error {
	var arr [3]any
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}
	t.Op = arr[0].(string)
	t.Key = strconv.Itoa(int(arr[1].(float64)))
	if arr[2] == nil {
		t.Val = nil
	} else {
		t.Val = new(int(arr[2].(float64)))
	}
	return nil
}

func (t txn) MarshalJSON() ([]byte, error) {
	key, _ := strconv.Atoi(t.Key)
	return json.Marshal([]any{t.Op, key, t.Val})
}
