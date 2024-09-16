---
id: Map Reduce
aliases: []
tags: []
---
The MapReduce model is inspired by functional programming concepts, particularly the **map** and **reduce** functions. It allows for the processing of large data sets by distributing the work across many machines in a cluster. The model is particularly useful for processing data that can be divided into independent chunks.

This model allows us to perform batch processing of big data sets

Works with data that is already in [[HDFS]].

## Main advantages
1. Can run arbitrary code, just define mapper and reducer
2. Runs computations on the same nodes that hold the data, this provides data locality
3. Failed mappers or reducers are restored independently.
## An Abstract view of a MapReduce job – word count

Input1 -> Map -> a,1 b,1  
Input2 -> Map -> b,1  
Input3 -> Map -> a,1 c,1  
| | |  
| | -> Reduce -> c,1  
| —–> Reduce -> b,2  
———> Reduce -> a,2

- input is (already) split into M files
- MR calls Map() for each input file, produces list of k,v pairs  
    “intermediate” data  
    each Map() call is a “task”
- when Maps are done,  
    MR gathers all intermediate v’s for each k,  
    and passes each key + values to a Reduce call
- final output is set of <k,v> pairs from Reduce()s

![[Pasted image 20240911163839.png]]