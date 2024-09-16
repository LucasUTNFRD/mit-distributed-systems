**Lazy Space Allocation** prevents disk space from being wasted by only allocating storage as data is written, improving overall storage efficiency, particularly for small or slowly growing files.

Without lazy space allocation, creating a new chunk would immediately reserve 64 MB of disk space, even if only a few KB of data were written to the chunk. By allocating space lazily, GFS avoids wasting large amounts of disk space, especially for small files or files that grow slowly.

## How lazy allocation works?
- **Deferred Space Reservation**:
    - When a new chunk is created, GFS does not immediately allocate the full 64 MB on disk. The chunk is treated as a file on the underlying **Linux file system**, and only the necessary amount of space is allocated as data is written to that chunk.
- **Space is Allocated On-Demand**:
    - As clients write more data to the chunk, the chunkserver dynamically expands the chunk file on disk, allocating more space as needed until it reaches the maximum size of 64 MB.
    - This prevents large, empty chunks from taking up unnecessary disk space if the chunk never reaches its full capacity (e.g., in the case of small files).