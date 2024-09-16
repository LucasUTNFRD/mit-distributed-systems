## Architecture
![[Pasted image 20240914205637.png]]
> GFS consists of a single master server and multiple chunkservers. Files are divided into chunks, which are stored on multiple chunkservers to provide redundancy.
### Master Server 
 Manages metadata, including the file system namespace, mapping of files to chunks, and chunk replica locations. It coordinates system-wide activities like garbage collection, replication, and chunk placement. It also handles client requests for metadata but doesn't directly manage data reads or writes to avoid being a bottleneck.
### Chunkservers
Store the actual data in fixed-size chunks (64MB). Each chunk is replicated across multiple chunkservers for fault tolerance, usually with three replicas. Chunkservers handle data reads and writes, and communicate with the master to maintain consistency and handle chunk leases for mutations. Each of theses is typically a commodity Linux machine running a user-lever server process.
### Clients
Interact with the master for metadata and with chunkservers for data. Clients cache chunk locations and read/write data directly from chunkservers after getting location information from the master.

### Interactions 
Let explain briefly the interactions for a simple read.

#### Read operation
- **Client to Master**: The client first communicates with the **master server** to request the location of the chunk it needs. The client provides the filename and byte offset to the master, which then translates it to the corresponding chunk index.
- **Master to Client**: The master responds with the chunk’s metadata, including its globally unique **chunk handle** and the **locations of the chunk replicas** stored on different chunkservers.
- **Client to Chunkserver**: After receiving the chunk location, the client directly contacts the nearest or least-loaded **chunkserver** holding a replica of the requested chunk. The client reads data from the chunkserver without further involving the master until the next request.
- **Caching**: The client caches the chunk locations to reduce future interactions with the master, improving efficiency for subsequent reads from the same file

## Chunk Size
Chunk size is the key design parameter. GFS choose 64 MB chunkfiles (this mora than typical file system blocks which are 4 KB or 8 KB).

[[lazy space allocation]] allocation avoids wasting space due to internal fragmentation.

### Advantages
1. Reduce intereacton with the master
2. Efficient network usage
3. reduce size of metadata stored by the master

A large chunk size offers serveral important advantages. First one, it reduces the interaction with the master.
- Since each chunk is large, clients require fewer interactions with the master server to get metadata (chunk locations) during large sequential reads or writes. For example, a large file consisting of many MBs or GBs can be read with only a few metadata requests, minimizing load on the master.
- This is especially beneficial for workloads that involve large streaming reads or writes, where clients can work with a single chunk for an extended period.

Second, it provided efficient network utiilization :
- Clients typically establish a persistent **TCP connection** with the chunkservers that hold the chunk. A large chunk size allows clients to maintain fewer connections over a longer time, reducing network overhead and increasing throughput.
- A persistent connection for multiple reads and writes within the same chunk reduces latency and improves performance.

Third it reduces the size of metadata stored in the master
- Larger chunks mean fewer chunks per file, which in turn reduces the amount of metadata that the master needs to manage. This allows the master to keep all metadata **in memory**, which speeds up metadata access and simplifies the overall architecture.
- For example, each chunk requires less than 64 bytes of metadata in the master’s memory, so the master can handle millions of chunks efficiently.

## Metadata 
**metadata** refers to the information that the **master server** maintains about the file system, which helps in managing the location, structure, and status of files and chunks. It includes:

1. **File and Directory Namespace**: The hierarchical structure of files and directories, similar to a file path in a traditional file system (e.g., `/home/user/file.txt`).
    
2. **File-to-Chunk Mapping**: Information about which chunks make up a particular file and the order of those chunks.
    
3. **Chunk Location Information**: Metadata on which **chunkservers** store replicas of each chunk.
    
4. **Chunk Version Numbers**: Used to track and manage chunk consistency, especially after failures, ensuring the master knows the latest version of a chunk.
    
This metadata is stored **in memory** on the master server for fast access, and critical metadata changes are logged persistently to disk for reliability. It helps coordinate file operations without handling actual data transfers.

## Consistency model
The **consistency model** in **Google File System (GFS)** is designed to be **relaxed** to balance performance and simplicity in a distributed environment. It provides the following key features:

1. **Guaranteed Atomicity for Namespace Operations**: Operations like file creation, deletion, and renaming are handled by the master and are atomic, ensuring that they are consistent across the system.
2. **Consistency and Defined Regions**:
    - A **consistent region** means that all clients will see the same data from all chunk replicas.
    - A **defined region** means the data has been written completely and successfully; all clients will read the same and correct data.
3. **Handling Data Mutations**:
    - GFS primarily uses **appends** instead of overwrites, ensuring simpler consistency for large, append-heavy workloads.
    - When multiple clients write concurrently to the same region, the data may become **consistent but undefined**, meaning all replicas contain the same data, but the exact content may not reflect any specific client’s write due to interleaved fragments.
    - If a write fails, the region can become **inconsistent**, meaning clients may see different data across replicas.
4. **Record Append**: GFS supports an **atomic record append** operation, which guarantees that data from multiple clients can be appended to a file without additional synchronization, ensuring at least one copy of each record is written.

This relaxed consistency model allows GFS to optimize for large-scale, distributed workloads, prioritizing **high throughput** over strict consistency
## System interactions
The GFS was designed to minimize the master involvement in all operations.

![[Pasted image 20240914223426.png]]

 1. Client Requests Chunk Information:
- The client sends a request to the **master** asking which chunkserver holds the [[lease]] (i.e., the primary role) for the chunk it wants to write to, along with the **locations of the chunk replicas**.
2. Master Replies with Chunk Information:
- The master replies with the identity of the **primary chunkserver** (the one holding the lease) and the locations of the other **secondary chunkservers** that hold replicas of the chunk. The client caches this information for future use.
3. Client Sends Data to All Chunkservers:
- The client sends the data to **all chunkservers** (both primary and secondaries) that store replicas of the chunk. The data is buffered on each chunkserver but not yet written to disk.
 4. Client Sends Write Request to Primary:
- After all chunkservers acknowledge receiving the data, the client sends a **write request** to the **primary chunkserver**. This request specifies the data that was sent earlier.
 5. Primary Assigns Serial Number and Forwards Write:
- The primary chunkserver assigns a **serial number** to the write request to establish the order of writes. It then applies the write to its local chunk and forwards the write request to the **secondary chunkservers**, ensuring they all write the data in the same order.
 6. Secondary Chunkservers Acknowledge Completion:
- The secondary chunkservers apply the write operation in the order assigned by the primary and send an acknowledgment back to the primary chunkserver once the write is complete.
 7. Primary Acknowledges Write to Client:
- Once all the secondaries have confirmed that they successfully wrote the data, the primary sends a final acknowledgment to the client, indicating that the write was successful.