# 6.5840 2024 Lecture 4: Consistency and Linearizability

## Topic: Consistency Models, Specifically Linearizability

### Introduction
- We need to reason about correct behavior for network services
  - Example: What application programmers can expect from GFS or Lab 2
  - This leads us to consistency models

### What's a Consistency Model?
- A specification for the relationship of different clients' views of a service
- Focus on key/value storage with network clients
  - `put(k, v) -> <done>`
  - `get(k) -> v`
- Question: Given some put/get calls, what outcome(s) are valid?

### Why is This Complex?
In ordinary programming, a read yields the last value written. However, in distributed systems, several factors can cause doubt about correct behavior:

- Concurrent reads/writes
- Replicas
- Caches
- Failure and recovery
- Lost messages
- Retransmission

### Why Does a Storage System Need a Formal Consistency Model?
1. For applications:
   - Hard to be correct without guarantees from storage
   - Example:
     ```
     Producer:
       put("result", 27)
       put("done", true)
     
     Consumer:
       while get("done") != true
         pause
       v = get(result)
     ```
   - Question: Is v guaranteed to be 27, no matter what?

2. For services:
   - Hard to design/implement/optimize without a specification
   - Example: Is it OK for clients to read from GFS replicas (rather than primary)?

### Types of Consistency Models
- Driven by desire to simplify application programmers' lives
- Sometimes codifying behavior convenient for implementors
- Overlapping definitions from different fields (e.g., FS, databases, CPU memory)
- Today's focus: Linearizability
- Other models we'll see:
  - Eventual consistency
  - Causal consistency
  - Fork consistency
  - Serializability
- Driving force: Performance / convenience / robustness tradeoffs

## Linearizability

### Definition and Importance
- A specification -- a requirement for how a service must behave
- Usually what people mean by "strong consistency"
- Matches programmer intuitions reasonably well
- Rules out many optimizations
- Will be implemented in Lab 2 and Lab 4 (with fault tolerance)

### Starting Point: Serial Specification
```python
db = {}
def put(k, v):
    db[k] = v
    return True
def get(k):
    return db[k]
```

### Concurrent Client Operations
- Client sends a request
- Request travels through network
- Server computes, talks to replicas, etc.
- Reply moves through network
- Client receives reply
- Other clients may send/receive/be waiting during this time

### Definition: A History
- Describes a timeline of possibly-concurrent operations
- Each operation has invocation and response times (RPC)
- Includes argument and return values
- Example:
  ```
  C1: |-Wx1-| |-Wx2-|
  C2:   |---Rx2---|
  ```
  - x-axis is real time
  - |- indicates client sent request
  - -| indicates client received reply
  - "Wx1" means "write value 1 to record x" (put(x, 1))
  - "Rx1" means "a read of record x yielded value 1" (get(x) -> 1)

### Linearizability Definition
A history is linearizable if:
1. You can find a point in time for each operation between its invocation and response, AND
2. The history's result values are the same as serial execution in that point order

### Examples and Analysis

### Implications of Linearizability
- Reads must return fresh data
- All clients must see writes in the same order
- Duplicate requests from retransmissions must be suppressed

### Implementing Linearizability
- Single serial server (non-crashing)
- Primary/backup replication for high availability

### Performance Considerations
- Bad news:
  - Serial aspect may limit parallel speedup
  - Replication requires lots of communication
  - Replication limits fault tolerance (replicas must be reachable)
- Good news:
  - You can shard if keys are independent

## Other Consistency Models

### Eventual Consistency
- A weak model
- Multiple copies of data (e.g., in different datacenters)
- Reads and writes consult any one replica (usually closest)
- Replicas synchronize updates in the background
- Popular (e.g., Amazon's Dynamo, Cassandra)
- Faster and more available than linearizability
- Drawbacks:
  - Reads may see stale data
  - Writes may appear out of order
  - Different clients may see different data
  - Concurrent writes need resolution
  - Cannot support operations like test-and-set

### General Pattern
You can usually choose only one of:
1. Strong consistency
2. Maximum availability

But not both.