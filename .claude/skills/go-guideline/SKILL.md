---
title: Google Go Style Guide
description: A canonical set of Go programming style rules, distilled from the Google Go Style Guide
---

This document defines **prescriptive Go style rules**. Follow these rules when writing or generating Go code unless explicitly instructed otherwise

# Naming

## Underscores

- Go identifiers **should not contain underscores**
- Allowed exceptions:
    1. Package names imported only by generated code
    2. `Test`, `Benchmark`, and `Example` functions in `*_test.go`

## Package Names

- Package names must be concise and **use only lowercase letters and numbers**
- Multi-word package names should remain unbroken and in all lowercase
- Package names are singular, unless they conflict with predefined keywords. **DO NOT** name a package `flags`; **DO** name a package `flag` instead

## Function and Method Names

### Avoid Package Name Repetition

Do not repeat the package name in exported identifiers.

```go
// BAD:
package yamlconfig
func ParseYAMLConfig(input string) (*Config, error)

// GOOD:
package yamlconfig
func Parse(input string) (*Config, error)
```

### Naming by Behavior

- **Return values** -> **noun-like names**
- **Actions** -> **verb-like names**
- Avoid the `Get` prefix

```go
// GOOD:
func (c *Config) JobName(key string) (string, bool)

// BAD:
func (c *Config) GetJobName(key string) (string, bool)
```

### Disambiguation

Add detail only when necessary.

```go
func (c *Config) WriteTextTo(w io.Writer) (int64, error)
func (c *Config) WriteBinaryTo(w io.Writer) (int64, error)
```

### Type-Specific Variants

If functions differ only by type, suffix the type name.

```go
func ParseInt(string) (int, error)
func ParseInt64(string) (int64, error)
```

If one version is primary, omit the type suffix.

```go
func (c *Config) Marshal() ([]byte, error)
func (c *Config) MarshalText() (string, error)
```

## Receiver Names

Receiver variable names must be:

- Short (usually one or two letters in length)
- Abbreviations for the type itself
- Applied consistently to every receiver for that type

## Initialisms

Use consistent casing for initialisms.

| Usage | Exported | Unexported |
| ----- | -------- | ---------- |
| ID    | `ID`     | `id`       |
| XML   | `XML`    | `xml`      |
| GRPC  | `GRPC`   | `gRPC`     |
| DB    | `DB`     | `db`       |
| DDoS  | `DDoS`   | `ddos`     |

## Avoid Repetition

Package names are always visible - avoid redundant naming

**Examples**

- `widget.NewWidget` -> `widget.New`
- `db.LoadFromDatabase` -> `db.Load`
- `myteampb.MyTeamMethodRequest` -> `myteampb.MethodRequest`

# Package Design

## Avoid Generic (util-like) Packages

Avoid package names like: `util`, `common`, `helper`, `models`, `types`, `domain`, `service`

Name packages after **what they provide**, NOT **what they contain**

```go
// BAD:
package util
func NewStringSet(...string) map[string]bool

// GOOD:
package stringset
type Set map[string]bool
func New(...string) Set
func (s Set) Sort() []string
```

## Package Size

Keep related types in the same package

Prefer **fewer, cohesive packages** over many tiny ones

**Examples**:

- Small packages that contain one cohesive idea that warrant nothing more being added nor nothing being removed:
    - package `csv`: CSV data encoding and decoding with responsibility split respectively between `reader.go` and `writer.go`
    - package `expvar`: whitebox program telemetry all contained in `expvar.go`

- Moderately sized packages that contain one large domain and its multiple responsibilities together:
    - package `flag`: command line flag management all contained in `flag.go`

- Large packages that divide several closely related domains across several files:
    - package `http`: the core of HTTP: `client.go`, support for HTTP clients; `server.go`, support for HTTP servers; `cookie.go`, cookie management
    - package `os`: cross-platform operating system abstractions: `exec.go`, subprocess management; `file.go`, file management; `tempfile.go`, temporary files

## Keep Types Close

Define types near where they are used.

## Organize by Responsibility

Group code by **behavior** and **functional responsibility**, not by abstraction level

```go
// BAD:
package model

// User represents a user in the system.
type User struct {...}

// GOOD:
package mngtservice

// User represents a user in the system.
type User struct {...}
func UsersByQuery(ctx context.Context, q *Query) ([]*User, *Iterator, error)
func UserIDByEmail(ctx context.Context, email string) (int64, error)
```

# Function Arguments

Where a function requires many inputs, consider using **Option Struct** or **Variadic Options**

## Option Struct

Use when:

- All callers need to specify one or more of the options
- A large number of callers need to provide many options
- The options are shared between multiple functions that the user will call

```go
type ReplicationOptions struct {
    Config              *replicator.Config
    PrimaryRegions      []string
    ReadonlyRegions     []string
    ReplicateExisting   bool
}
func EnableReplication(ctx context.Context, opts ReplicationOptions) { // ...}
```

## Variadic Options

Use when:

- Most callers will not need to specify any options
- Most options are used infrequently
- There are a large number of options
- Options require arguments
- Options could fail or be set incorrectly (in which case the option function returns an error)
- Options require a lot of documentation that can be hard to fit in a struct
- Users or other packages can provide custom options

```go
type replicationOptions struct {
    readonlyCells       []string
    replicateExisting   bool
    overwritePolicies   bool
    healthWatcher       health.Watcher
}

type ReplicationOption func(*replicationOptions)

func ReadonlyCells(cells ...string) ReplicationOption {
    return func(opts *replicationOptions) {
        opts.readonlyCells = append(opts.readonlyCells, cells...)
    }
}
func ReplicateExisting(enabled bool) ReplicationOption {
    return func(opts *replicationOptions) {
        opts.replicateExisting = enabled
    }
}

var DefaultReplicationOptions = []ReplicationOption{
    OverwritePolicies(true),
    ReplicationInterval(12 * time.Hour),
    CopyWorkers(10),
}

func EnableReplication(ctx context.Context, config *placer.Config, primaryCells []string, opts ...ReplicationOption) {
    var options replicationOptions
    for _, opt := range DefaultReplicationOptions {
        opt(&options)
    }
    for _, opt := range opts {
        opt(&options)
    }
}
```

# Errors

## Error Messages

Avoid redundant phrases like "failed to"

```go
// BAD:
return fmt.Errorf("failed to create new store: %w", err)

// GOOD:
return fmt.Errorf("new store: %w", err)
```

## Indent Error Flow (Happy Path)

Handle errors early and avoid `else` blocks

```go
if err != nil {
    return err
}
// normal execution
```

# Interfaces

- **DO** **generally** define interfaces in the **consumer** package
- **DO** **generally** return concrete types from producers
- **DO NOT** define interfaces before they are used
- **DO NOT** use interface-typed parameters if the users of the package do not need to pass different types for them
- **DO NOT** export interfaces that the users of the package do not need

# Method Receivers

Use a **pointer receiver** when:

- The method needs to mutate the receiver
- The receiver is a struct containing fields that cannot safely be copied
- The receiver is a "large" struct or array
- The receiver is a struct or array, any of whose elements is a pointer to something that may be mutated

Use a **value receiver** when:

- The receiver is a slice and the method doesn't reslice or reallocate the slice
- The receiver is a built-in type, such as an integer or a string, that does not need to be modified
- The receiver is a map, function, or channel
- The receiver is a "small" array or struct that is naturally a value type with no mutable fields and no pointers

When in doubt, use a pointer receiver

# Tests

- Prefer `t.Context()` over `context.Background()` or `context.TODO()` within tests
- When initializing a logger for a test, use `t.Output()` as the destination. This ensures log output is captured and associated with the specific test failure
- Every helper function **MUST** call `t.Helper()` as its first line

## Error Testing

- Do not perform string comparisons on error messages. Error strings are for humans, not for control flow, and are subject to change
- Verify errors only via sentinel values or specific types. Only check for specific errors if the package explicitly defines them
- If no sentinel/type exists, only check that error is not `nil`

# Contexts

- Do not store `context.Context` in structs
- Pass context explicitly to methods
- In tests, prefer `testing.TB.Context()`

# Documentation

- All top-level exported names must have doc comments
- Unexported types or functions with non-obvious behavior or meaning should also have doc comments
- Each package must have a single package comment
- Comments should be full sentences ending with a period, that start with the name of the object being described; an article may precede the name

# References

- [Go Style Decisions](https://github.com/google/styleguide/blob/gh-pages/go/decisions.md)
- [Go Style Best Practices](https://github.com/google/styleguide/blob/gh-pages/go/best-practices.md)
- [Package names](https://go.dev/blog/package-names)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Contexts and structs](https://go.dev/blog/context-and-structs)
- [Style guideline for Go packages](https://rakyll.org/style-packages/)
