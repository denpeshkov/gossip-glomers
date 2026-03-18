---
name: sql-guideline
description: This document defines the SQL style guideline to be followed when writing SQL code, both queries (DML) and migrations (DDL)
---

# SQL Style Guidelines

## General Principles

- All SQL must be written in lowercase using `snake_case` for database objects names
- Avoid implicit behavior; Always prefer explicit definitions
- Favor long-term maintainability over short-term convenience

## Naming Conventions

- Use **singular nouns** for table names

**Good**

```sql
create table member (...);
```

**Bad**

```sql
create table members (...);
```

## Constraints

Constraints must be explicitly defined and follow consistent naming

The table below describes the naming conventions for custom PostgreSQL constraints:

| Type                     | Syntax                                                                                            |
| ------------------------ | ------------------------------------------------------------------------------------------------- |
| **Primary Key**          | `pk_<table name>`                                                                                 |
| **Foreign Key**          | `fk_<table name>_<column name>[_and_<column name>]*_<foreign table name>`                         |
| **Index**                | `index_<table name>_on_<column name>[_and_<column name>]*[_and_<column name in partial clause>]*` |
| **Unique Constraint**    | `unique_<table name>_<column name>[_and_<column name>]*`                                          |
| **Check Constraint**     | `check_<table name>_<column name>[_and_<column name>]*[_<validation_type>]?`                      |
| **Exclusion Constraint** | `excl_<table name>_<column name>[_and_<column name>]*_[_<exclusion_type>]?`                       |
