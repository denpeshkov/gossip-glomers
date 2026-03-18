---
name: postgresql-guideline
description: PostgreSQL performance optimization and best practices. Use this skill when writing, reviewing, or optimizing Postgres queries, schema designs, or database configurations
---

# PostgreSQL Best Practices

Comprehensive performance optimization guide for PostgreSQL

## When to Apply

Reference these guidelines when:

- Writing SQL queries or designing schemas
- Implementing indexes or query optimization
- Reviewing database performance issues
- Optimizing for Postgres-specific features

## Rule Categories by Priority

| Category                   | Impact      | Prefix      |
| -------------------------- | ----------- | ----------- |
| Query Performance          | CRITICAL    | `query-`    |
| Schema Design              | HIGH        | `schema-`   |
| Concurrency and Locking    | MEDIUM-HIGH | `lock-`     |
| Data Access Patterns       | MEDIUM      | `data-`     |
| Monitoring and Diagnostics | LOW-MEDIUM  | `monitor-`  |
| Advanced Features          | LOW         | `advanced-` |

## How to Use

Read individual rule files for detailed explanations and SQL examples:

```
rules/query-missing-indexes.md
rules/schema-partial-indexes.md
```

Each rule file contains:

- Brief explanation of why it matters
- Incorrect SQL example with explanation
- Correct SQL example with explanation
- Optional `EXPLAIN` output or metrics
- Additional context and references
