package kvsrv

import (
	"log"
	"sync"
)

const Debug = true

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type dupTable struct {
	seq   int
	value string
}

type KVServer struct {
	mu          sync.Mutex
	data        map[string]string
	clientTable map[int64]*dupTable
	// Your definitions here.
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	// duplicate detection
	// if client is new, create the map of clientid -> duplicateTable
	if kv.clientTable[args.ClientID] == nil {
		reply.Value = kv.data[args.Key]
		return
	}

	dt := kv.clientTable[args.ClientID]

	// if seq exists
	if dt.seq == args.SeqNum {
		reply.Value = dt.value
		return
	}

	dt.seq = args.SeqNum
	dt.value = kv.data[args.Key]
	reply.Value = dt.value
	// value, ok := kv.data[args.Key]
	// if ok {
	// 	reply.Value = value
	// } else {
	// 	reply.Value = ""
	// }
}

// Put(key, value) installs or replaces the value for a particular key in the map
func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	// duplicate detection
	// if client is new, create the map of clientid -> duplicateTable
	if kv.clientTable[args.ClientID] == nil {
		kv.data[args.Key] = args.Value
		return
		// kv.clientTable[args.ClientId] = &dupTable{-1, ""}
	}

	dt := kv.clientTable[args.ClientID]

	// if seq exists
	if dt.seq == args.SeqNum {
		return
	}

	dt.seq = args.SeqNum
	dt.value = ""
	kv.data[args.Key] = args.Value
	//	kv.data[args.Key] = args.Value
	// reply.Value = ""
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// oldValue, _ := kv.data[args.Key]
	// kv.data[args.Key] = oldValue + args.Value
	// reply.Value = oldValue
	// duplicate detection
	// if client is new, create the map of clientid -> duplicateTable
	if kv.clientTable[args.ClientID] == nil {
		kv.clientTable[args.ClientID] = &dupTable{-1, ""}
	}

	dt := kv.clientTable[args.ClientID]
	DPrintf("Server: ClientId %v Append reply %v and dt.seq is %v\n",
		args.ClientID, args.SeqNum, dt.seq)

	// if seq exists
	if dt.seq == args.SeqNum {
		reply.Value = dt.value
		return
	}

	// return old value
	dt.seq = args.SeqNum
	dt.value = kv.data[args.Key]
	reply.Value = dt.value
	// append map[key]
	kv.data[args.Key] = kv.data[args.Key] + args.Value
}

func StartKVServer() *KVServer {
	kv := new(KVServer)

	// You may need initialization code here.
	kv.data = make(map[string]string)
	kv.clientTable = make(map[int64]*dupTable)

	return kv
}
