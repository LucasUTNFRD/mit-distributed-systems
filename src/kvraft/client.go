package kvraft

import (
	"crypto/rand"
	"math/big"
	"sync/atomic"

	"6.5840/labrpc"
)

type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	clientId int64
	seqNum   uint64 // field to ensure idempotent
	leaderID int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	// You'll have to add code here.
	ck.seqNum = 0
	ck.clientId = nrand()
	ck.leaderID = 0 //assume leader is 0
	return ck
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string) string {
	// You will have to modify this function.

	args := GetArgs{
		Key:      key,
		ClientID: atomic.LoadInt64(&ck.clientId),
		SeqNum:   atomic.AddUint64(&ck.seqNum, 1),
	}
	for {
		var reply GetReply
		DPrintf("Clerk %d: Sending Get request to server %d with args %+v", ck.clientId, ck.leaderID, args)
		ok := ck.servers[ck.leaderID].Call("KVServer.Get", &args, &reply)
		if ok && reply.Err == OK {
			DPrintf("Clerk %d: Received Get reply from server %d with value %s", ck.clientId, ck.leaderID, reply.Value)
			return reply.Value
		}
		if ok && reply.Err == ErrNoKey {
			DPrintf("Clerk %d: Received Get reply from server %d with ErrNoKey", ck.clientId, ck.leaderID)
			return ""
		}
		for server := range ck.servers {
			if server != ck.leaderID {
				DPrintf("Clerk %d: Retrying Get request to server %d with args %+v", ck.clientId, server, args)
				ok := ck.servers[server].Call("KVServer.Get", &args, &reply)
				if ok && reply.Err == OK {
					ck.leaderID = server
					DPrintf("Clerk %d: Received Get reply from server %d with value %s", ck.clientId, server, reply.Value)
					return reply.Value
				}
				if ok && reply.Err == ErrNoKey {
					ck.leaderID = server
					DPrintf("Clerk %d: Received Get reply from server %d with ErrNoKey", ck.clientId, server)
					return ""
				}
			}
		}
	}
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.
	args := PutAppendArgs{
		Key:      key,
		Value:    value,
		ClientID: atomic.LoadInt64(&ck.clientId),
		SeqNum:   atomic.AddUint64(&ck.seqNum, 1),
		Op:       op,
	}
	for {
		var reply PutAppendReply
		DPrintf("Clerk %d: Sending %s request to server %d with args %+v", ck.clientId, op, ck.leaderID, args)
		ok := ck.servers[ck.leaderID].Call("KVServer."+op, &args, &reply)
		if ok && reply.Err == OK {
			DPrintf("Clerk %d: Received %s reply from server %d", ck.clientId, op, ck.leaderID)
			return
		}
		for server := range ck.servers {
			if server != ck.leaderID {
				DPrintf("Clerk %d: Retrying %s request to server %d with args %+v", ck.clientId, op, server, args)
				ok := ck.servers[server].Call("KVServer."+op, &args, &reply)
				if ok && reply.Err == OK {
					ck.leaderID = server
					DPrintf("Clerk %d: Received %s reply from server %d", ck.clientId, op, server)
					return
				}
			}
		}
	}

}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
