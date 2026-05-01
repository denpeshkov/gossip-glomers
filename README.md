# Gossip Glomers

Solutions to [Gossip Glomers](https://fly.io/dist-sys/), a series of distributed systems challenges brought by Fly.io and Kyle Kingsbury

# Challenge #2: Unique ID Generation

My solution uses UUIDs to ensure global uniqueness.

Alternative approaches:

- ULID
- Snowflake: Or other k-ordered ID algorithms
- Each node generates IDs in the format `{node_id}-{local_counter}` to avoid collisions without coordination

# Challenge #3d: Efficient Broadcast, Part I

This solution uses a full-mesh topology. When a node receives a `broadcast` request from a client, it immediately sends that message to every other node in the cluster

To achieve resilient delivery, the node uses goroutines trying to send the request until the ack is received

Because this specific challenge assumes that node can't fail, we do not need to re-broadcast (gossip) the request from peer to peer. This limits the total traffic for a single client broadcast to O(n) messages

# Challenge #3e: Efficient Broadcast, Part II

Almost the same solution as the previous one. Upon receiving a `broadcast` request, the node simply appends the message to a pending buffer

A background gossip goroutine periodically flushes this buffer, sending all accumulated messages to peers in a single batch.

# Challenge #4: Grow-Only Counter

## Solution using shared counter and CAS

Since we are using a sequentially-consistent KV store, there is no global total order; only the order of operations for each individual node is preserved

To perform an `add` request, each node executes a CAS loop to update the shared counter value

The final value is the maximum observed across all nodes. Since the counter is grow-only, we know that the "most correct" value is always the largest one, as it represents the state that has incorporated the most increments

## Solution using CRDTs

Since we are using a sequentially-consistent KV store, there is no global total order; only the order of operations for each individual node is preserved

This approach uses a G-Counter CRDT where each node maintains its own counter in a KV store
The total value is the sum of all counters, eliminating the need for a global shared counter and CAS operation

# Challenge #5a: Single-Node Kafka-Style Log

Store message logs in a local `map`, using the log name as the key

Store committed offsets in a separate local `map` keyed by the log name

Both maps are protected by the mutex

# Challenge #5b: Single-Node Kafka-Style Log

Each log is stored in a linearizable KV store using the key format `log_{key}`

Committed offsets are stored in the same linearizable KV store using the key format `committed_offsets_{key}`

Since multiple clients are accessing the same keys concurrently, we use a CAS operation within a retry loop to ensure atomic updates to both logs and offsets

# Challenge #5c: Efficient Kafka-Style Log

Each log is stored in a linearizable KV store using the key format `log_{key}`

Committed offsets are stored in the same linearizable KV store using the key format `committed_offsets_{key}`

In the previous challenge, we used CAS to address concurrency issues when writing to the KV store
In this challenge, requests are routed based on the hash of the key to the same node (primary), ensuring that there are no concurrent updates to the same key
Only one node performs updates, and we use a mutex to synchronize local goroutines

# Challenge #6a: Single-Node, Totally-Available Transactions

Use a `map` protected by lock to serialize every transaction agains the k/v store
