package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"6.5840/labgob"
	"6.5840/labrpc"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type ServerState int

const (
	Follower ServerState = iota
	Candiate
	Leader
	Dead
)

type LogEntry struct {
	Command interface{}
	Term    int
	Index   int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).

	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	lastReceive time.Time

	state ServerState
	// persistent
	currentTerm int
	votedFor    int
	log         []LogEntry
	// volatile
	commitIndex int
	lastApplied int
	// volatile leaders
	nextIndex  []int
	matchIndex []int

	applyCh chan ApplyMsg
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var term int
	var isleader bool
	// Your code here (3A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.currentTerm
	isleader = rf.state == Leader
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)

	// 3A Code
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	raftstate := w.Bytes()
	rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).
}

// TODO
// Implement leader logic
// Implement AppendEntriesRPC

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	// 3A
	Term         int // candidate terms
	CandidateId  int // candidate requesting vote
	LastLogIndex int // index of candidate last log entry
	LastLogTerm  int // term of candidates last log entry
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int  // currentTerm, for candidate update itself.
	VoteGranted bool // true means candidate received bool
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}
	if args.Term > rf.currentTerm {
		rf.convertToFollower(args.Term)
	}
	logEntry := rf.getLastLogEntry()
	upToDate := false
	if args.LastLogTerm > logEntry.Term {
		upToDate = true
	} else if args.LastLogTerm == logEntry.Term && args.LastLogIndex >= logEntry.Index { // break tie by comparing indices
		upToDate = true
	}
	// gran vote if
	// 1.  hasn’t already voted for another candidate in the current term
	// 2. candidate’s log is at least as up-to-date as the server’s log
	if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && upToDate {
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
		// Reset election timeout
		rf.lastReceive = time.Now()
		// rf.persist() // Save vote decision to persistent storage
	} else {
		reply.VoteGranted = false
	}
	reply.Term = rf.currentTerm
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.

type AppendEntriesArgs struct {
	Term         int // the leader's current term number.
	LeaderID     int // The ID of the leader, allowing followers to redirect clients if necessary.
	PrevLogIndex int // The index of the log entry immediately preceding the new entries being sent. This is crucial for the consistency check.
	PrevLogTerm  int // The term number of the prevLogIndex entry, again essential for consistency.
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) applyLogs() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
		msg := ApplyMsg{
			CommandValid: true,
			Command:      rf.log[i-1].Command,
			CommandIndex: i,
		}
		rf.applyCh <- msg
		rf.lastApplied = i
	}
}

// func (rf *Raft) sendAppendEntries(
// 	server int,
// 	args *AppendEntriesArgs,
// 	reply *AppendEntriesReply,
// ) bool {
// 	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
// 	return ok
// }

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	if args.Term > rf.currentTerm || rf.state == Follower {
		rf.convertToFollower(args.Term)
	}

	rf.lastReceive = time.Now() // Reset election timeout

	// Check if log contains an entry at prevLogIndex with prevLogTerm
	if args.PrevLogIndex > rf.getLastLogEntry().Index ||
		(args.PrevLogIndex > 0 && rf.log[args.PrevLogIndex-1].Term != args.PrevLogTerm) {
		return
	}

	// Append new entries
	for i, entry := range args.Entries {
		index := args.PrevLogIndex + 1 + i
		if index > len(rf.log) {
			rf.log = append(rf.log, entry)
		} else if rf.log[index-1].Term != entry.Term {
			rf.log = rf.log[:index-1]
			rf.log = append(rf.log, entry)
		}
	}

	// Update commit index
	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = min(args.LeaderCommit, rf.getLastLogEntry().Index)
		go rf.applyLogs()
	}

	reply.Success = true
}

func (rf *Raft) sendAppendEntries(
	server int,
	args *AppendEntriesArgs,
	reply *AppendEntriesReply,
) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) Start(command interface{}) (int, int, bool) {
	// index := -1
	// term := -1
	// isLeader := true

	// Your code here (3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != Leader {
		return -1, rf.currentTerm, false
	}

	term := rf.currentTerm
	index := rf.getLastLogEntry().Index
	rf.log = append(
		rf.log,
		LogEntry{Command: command, Term: term, Index: index},
	)

	return index, term, true
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.state = Dead
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) getElectionTimeout() time.Duration {
	ms := 50 + (rand.Int63() % 300)
	return time.Duration(ms) * time.Millisecond
}

// TODO implement Leader Election

// On conversion to candidate, start election
// Increment the current term
// Vote for self
// Reset election timer
// Send requestvote RPC to all servers

func (rf *Raft) convertToCandidate() {
	rf.state = Candiate
	rf.currentTerm++
	rf.votedFor = rf.me
	rf.lastReceive = time.Now()
	DPrintf("[%d]Attemptiing election at term %d", rf.me, rf.currentTerm)
}

func (rf *Raft) convertToFollower(term int) {
	rf.state = Follower // peer is passive
	rf.currentTerm = term
	// rf.log = append(rf.log, LogEntry{Term:0})
	rf.votedFor = -1
	rf.lastReceive = time.Now()
	DPrintf("[%d]Reverted to follower at term %d", rf.me, term)
}

func (rf *Raft) convertToLeader() {
	rf.state = Leader
	lastLogEntry := rf.getLastLogEntry()
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	for i := range rf.peers {
		rf.nextIndex[i] = lastLogEntry.Index + 1
		rf.matchIndex[i] = 0
	}
	DPrintf("Peer %d became leader for term %d", rf.me, rf.currentTerm)
}

func (rf *Raft) getLastLogEntry() LogEntry {
	if len(rf.log) == 0 {
		// Return a default LogEntry if the log is empty
		return LogEntry{Term: 0, Index: 0}
	}
	return rf.log[len(rf.log)-1]
}

func (rf *Raft) KickOffElection() {
	rf.mu.Lock()
	rf.convertToCandidate()
	lastLogEntry := rf.getLastLogEntry()
	term := rf.currentTerm
	args := RequestVoteArgs{
		Term:         term,
		CandidateId:  rf.me,
		LastLogIndex: lastLogEntry.Index,
		LastLogTerm:  lastLogEntry.Term,
	}
	// rf.persist()
	rf.mu.Unlock()
	DPrintf("[%d] Starting election for term %d", rf.me, term)
	// this is the most beatiful use case for channels i have ever seen xd
	voteCh := make(chan bool, len(rf.peers))
	for server := range rf.peers {
		if server == rf.me {
			voteCh <- true
			continue
		}
		go func(server int) {
			reply := RequestVoteReply{}
			DPrintf("[%d] Sending request vote to %d at term %d", rf.me, server, args.Term)
			ok := rf.sendRequestVote(server, &args, &reply)
			if !ok {
				voteCh <- false
				return
			}
			rf.mu.Lock()
			defer rf.mu.Unlock()
			if reply.Term > rf.currentTerm {
				// convert to follower
				rf.convertToFollower(reply.Term)
				// rf.persist()
				voteCh <- false
				return
			}
			if reply.VoteGranted {
				voteCh <- true
			} else {
				voteCh <- false
			}
		}(server)
	}

	rf.countVotes(voteCh, term)
	// idk, what should i do here?
	// collect votes and do the counting
}

const HeartBeatInterval = 100 * time.Millisecond

func (rf *Raft) leaderLoop() {
	for !rf.killed() {
		rf.mu.Lock()
		if rf.state != Leader {
			rf.mu.Unlock()
			return
		}
		rf.sendHeartbeats()
		rf.mu.Unlock()
		time.Sleep(HeartBeatInterval)
	}
}

func (rf *Raft) sendHeartbeats() {
	for i := range rf.peers {
		if i != rf.me {
			go rf.sendHeartbeat(i)
		}
	}
}

func (rf *Raft) sendHeartbeat(server int) {
	rf.mu.Lock()
	prevLogIndex := rf.nextIndex[server] - 1
	prevLogTerm := 0
	if prevLogIndex > 0 {
		prevLogTerm = rf.log[prevLogIndex-1].Term
	}
	entries := rf.log[prevLogIndex:]
	args := &AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderID:     rf.me,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: rf.commitIndex,
	}
	rf.mu.Unlock()

	reply := &AppendEntriesReply{}
	ok := rf.sendAppendEntries(server, args, reply)

	if ok {
		rf.mu.Lock()
		defer rf.mu.Unlock()

		if reply.Term > rf.currentTerm {
			rf.convertToFollower(reply.Term)
			return
		}

		if rf.state != Leader || rf.currentTerm != args.Term {
			return
		}

		if reply.Success {
			rf.nextIndex[server] = prevLogIndex + len(entries) + 1
			rf.matchIndex[server] = rf.nextIndex[server] - 1
			rf.updateCommitIndex()
		} else {
			rf.nextIndex[server] = max(1, rf.nextIndex[server]-1)
		}
	}
}

// countVotes collects votes from the vote channel and determines whether the candidate wins the election.
func (rf *Raft) countVotes(voteCh chan bool, term int) {
	votes := 0 // voted for self
	majority := len(rf.peers)/2 + 1
	// we need the term were we start counting votes
	for range rf.peers {
		if <-voteCh {
			votes++
		}
		// A candidate wins an election if it receives votes from a majority
		if votes >= majority {
			rf.mu.Lock()
			// the win must ocurr in the same term.
			if rf.state == Candiate && rf.currentTerm == term {
				rf.convertToLeader()
				// this shold be in convertToLeader method
				go rf.leaderLoop()
			}
			rf.mu.Unlock()
			return
		}
	}
	// election failed
	rf.mu.Lock()
	if rf.state == Candiate && rf.currentTerm == term {
		rf.convertToFollower(rf.currentTerm)
	}
	rf.mu.Unlock()
}

func (rf *Raft) ticker() {
	for !rf.killed() {
		time.Sleep(rf.getElectionTimeout())

		rf.mu.Lock()
		if rf.state != Leader && time.Since(rf.lastReceive) >= rf.getElectionTimeout() {
			rf.mu.Unlock()
			rf.KickOffElection()
		} else {
			rf.mu.Unlock()
		}
	}
}

func (rf *Raft) updateCommitIndex() {
	for n := rf.commitIndex + 1; n <= rf.getLastLogEntry().Index; n++ {
		if rf.log[n-1].Term == rf.currentTerm {
			count := 1
			for i := range rf.peers {
				if i != rf.me && rf.matchIndex[i] >= n {
					count++
				}
			}
			if count > len(rf.peers)/2 {
				rf.commitIndex = n
				go rf.applyLogs()
			}
		}
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg,
) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (3A, 3B, 3C).
	// 3A initialization
	currentTerm := 0
	rf.convertToFollower(currentTerm)

	rf.log = []LogEntry{}

	// Initialize volatile state
	rf.commitIndex = 0
	rf.lastApplied = 0

	// Initialize leader state
	// rf.log = []LogEntry{{Term: 0, Index: 0}}
	rf.applyCh = applyCh
	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	DPrintf("%d intialized", rf.me)
	return rf
}
