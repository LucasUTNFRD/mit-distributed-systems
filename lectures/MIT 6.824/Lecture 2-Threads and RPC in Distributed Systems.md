#### Why Go?
- **Good for distributed systems**: Threads, garbage collection, and type safety.
- **Threads (goroutines)**: Go’s concurrency mechanism is simple and effective for parallelism in distributed systems.
- **RPC support**: Easy for client/server communication.

#### Threads Overview

- **Definition**: Threads are independent sequences of execution within a program.
- **Benefits**:
    - **I/O concurrency**: Handle multiple client requests, allowing non-blocking operations.
    - **Multicore performance**: Efficient use of multiple cores.
    - **Convenience**: Can handle background tasks like status checks.
- **Alternative**: Event-driven programming avoids thread costs but lacks multicore speedup and is more complex to code.

#### Threading Challenges

- **Race conditions**: Multiple threads accessing shared data can cause conflicts. Use synchronization tools like locks (Go’s `sync.Mutex`).
- **Coordination**: Threads producing and consuming data need synchronization via channels, `sync.Cond`, or `sync.WaitGroup`.
- **Deadlock**: Occurs when threads wait indefinitely on each other.

#### Web Crawler Example

- **Crawler challenges**:
    - **Concurrency**: Fetching multiple pages concurrently increases throughput.
    - **Avoid duplicates**: Track which URLs have been visited to avoid cycles and network strain.
    - **Completion detection**: Know when all pages have been crawled.

#### Three Web Crawler Solutions

1. **Serial Crawler**: Depth-first search, no concurrency—slow.
2. **Concurrent Mutex Crawler**: Uses threads, with shared `fetched` map and synchronization via `sync.Mutex`.
3. **Concurrent Channel Crawler**: Communication between threads using Go channels, no shared memory. Channels synchronize and communicate data.

#### Go Channels

- **Definition**: Channels are objects for communication between threads, allowing synchronization by blocking on send/receive.
- **Usage**: Good for tasks where threads need to communicate events (e.g., URL fetching completion).

#### RPC (Remote Procedure Call)

- **Purpose**: Simplify client/server communication in distributed systems.
- **Mechanism**:
    - Client sends a request; server processes and replies.
    - **Go RPC library**: Automatically handles marshaling (converting data into network format) and communication.

#### RPC Challenges

- **Failures**: Network issues can cause incomplete RPCs.
- **Handling failures**: "Best-effort RPC" retries requests, but may lead to duplicates.
- **At-most-once RPC**: Avoids duplicate processing by tracking request IDs, but requires handling complexity (like client crashes or retries).

#### Conclusion

- **Go's simplicity and concurrency** make it well-suited for distributed systems.
- **RPC and threads** are fundamental tools for building efficient, scalable systems. Understanding concurrency models like locks, channels, and failure handling in RPC is crucial for reliable distributed applications.