## LAB 4A
- Completed the implementation of the KVServer and Clerk for LAB 4A.
- Integrated Raft for consensus and replication in the key-value service.
- Added synchronization mechanisms to ensure linearizability and consistency.
- Implemented Get, Put, and Append RPC handlers in KVServer.
- Added debug print messages to trace the flow of execution and state changes.
- Ensured that the Clerk handles retries and leader changes gracefully.

Known Issues:
- Encountered random test failures due to issues in the Raft library.
- Specifically, the failures are related to the SendHeartBeat function.
- Observed index out of range errors and suboptimal timer handling in SendHeartBeat.

Next Steps:
- Optimize the timer in the SendHeartBeat function to improve performance and reliability.
- Handle index out of range errors in SendHeartBeat to prevent crashes and ensure robustness.
