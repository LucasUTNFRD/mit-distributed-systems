## Strong consistency
strong consistency as being a system whose behavior to app look like you would expect from talking to a single server.(ideal consistency model)

We study GFS because they provided a solution to bad replication desing.

## Notes about GFS 
- big,fast 
- global storage system 
- in order to get big and fast they split data , we call this sharding.
- automatic failure recovery.
- Single data center for the replicas. We dont want replicas far from each other.
- The GFS was for internal use.
	- Tailored for number of way to handle sequential number of W and R.
	- We talking about big sequential (not random) access.
- This paper does not guarantee that they get the right data. The hope is that they take advanage of that to get more performance.

## Master data 
filaname -> array of chunk handles (non volatile)
handle -> list of chunkservers (volatile).
	->  version # (non volatile).
	-> primary. (volatile).
	-> lease of expiration (volatile).
	-> all this is stored in disk and also is in memory.
log on disk,check point -> disk.
## Writes 
No primary on master 
find up to date replicas
up to date means that the master that hands out the latest  versions on disk.
pick primary and secondary servers.
Increment V# and write to disk.
tells p,s , V# . 

