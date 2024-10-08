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

// func max(a, b int) int {
// 	if a > b {
// 		return a
// 	}
// 	return b
// }

//TODO

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

const (
	HeartBeatTimeOut = 100
	ElectTimeOutBase = 450

	ElectTimeOutCheckInterval = time.Duration(250) * time.Millisecond
	CommitCheckTimeInterval   = time.Duration(100) * time.Millisecond
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

	// ----- PERSISTENT STATE -----

	//The current Term
	currentTerm int

	//Who was voted for in the most recent term
	votedFor int

	log []LogEntry

	// ----- VOLATILE STATE -----

	//Candidate,follower or leader
	state ServerState

	//index of highest log entry know to be commited
	commitIndex int

	//Index of highest log entre applied to state machine
	lastApplied int

	//index of the next log entry to send
	nextIndex []int

	//Highest log entry know to be replicated
	matchIndex []int

	applyCh     chan ApplyMsg
	lastReceive time.Time
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	return rf.currentTerm, rf.state == Leader
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

func (rf *Raft) isLogUpToDate(cLastIdx, cLastTerm int) bool {
	lastLogIndex, lastLogTerm := rf.getLastLogIndex(), rf.getLastLogTerm()

	if cLastTerm > lastLogTerm {
		return true
	}

	// 2. If the terms are equal, check the log index
	if cLastTerm == lastLogTerm && cLastIdx >= lastLogIndex {
		return true
	}

	// If neither condition is true, the candidate's log is not up-to-date
	return false
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.currentTerm
	reply.VoteGranted = false

	if args.Term < rf.currentTerm {
		return
	}
	if args.Term > rf.currentTerm {
		rf.convertToFollower(args.Term)
	}

	// Check if we've already voted in this term
	if rf.votedFor != -1 && rf.votedFor != args.CandidateId {
		rf.lastReceive = time.Now()
		return
	}

	// Check if the candidate's log is at least as up-to-date as ours
	// lastLogEntry := rf.getLastLogEntry()
	candidateLogUpToDate := rf.isLogUpToDate(args.LastLogIndex, args.LastLogTerm)

	if !candidateLogUpToDate {
		return
	}

	// If we reach here, we can grant the vote
	rf.votedFor = args.CandidateId
	reply.VoteGranted = true
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
	Term          int  // current term of the receiving node
	Success       bool // AppendEntry declined or accepted
	ConflictIndex int  // if declined, specifying the conflicting index
	ConflictTerm  int  // if declined, specifying the conflicting term
}

func (rf *Raft) Start(command interface{}) (int, int, bool) {
	// Your code here (3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != Leader {
		return -1, rf.currentTerm, false
	}

	term := rf.currentTerm
	entry := LogEntry{Term: term, Command: command}
	rf.log = append(rf.log, entry)

	return rf.getLastLogIndex(), term, true
}

func (rf *Raft) sendAppendEntries(serverTo int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[serverTo].Call("Raft.AppendEntries", args, reply)
	return ok
}

// AppendEntries handler
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		rf.mu.Unlock()
		reply.Success = false
		return
	}

	rf.lastReceive = time.Now()

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term // 更新iterm
		rf.votedFor = -1           // 易错点: 更新投票记录为未投票
		rf.state = Follower
	}

	if len(args.Entries) == 0 {
		DPrintf("[%v] received a heartbeat from leader %v: %+v\n", rf.me, args.LeaderID, args)
	} else {
		DPrintf("[%v] received AppendEntries from leader %v: %+v\n", rf.me, args.LeaderID, args)
	}

	if args.PrevLogIndex >= len(rf.log) || rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		// 2. Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)

		reply.Term = rf.currentTerm
		rf.mu.Unlock()
		reply.Success = false
		DPrintf("[%v] detected illegal parameters in heartbeat:\n\t args.PrevLogIndex=%v, args.PrevLogTerm=%v, \n\tlen(self.log)=%v, the term of the last position in self is: %v\n", rf.me, args.PrevLogIndex, args.PrevLogTerm, len(rf.log), rf.log[len(rf.log)-1].Term)
		return
	}
	// 3. If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it (§5.3)
	if len(args.Entries) != 0 && len(rf.log) > args.PrevLogIndex+1 && rf.log[args.PrevLogIndex+1].Term != args.Entries[0].Term {
		DPrintf("[%v] log conflicts with args, removing entries\n", rf.me)
		rf.log = rf.log[:args.PrevLogIndex+1]
	}

	// 4. Append any new entries not already in the log
	rf.log = append(rf.log, args.Entries...)
	if len(args.Entries) != 0 {
		DPrintf("[%v] succesfull append operation\n", rf.me)
	}

	reply.Success = true
	reply.Term = rf.currentTerm

	if args.LeaderCommit > rf.commitIndex {
		// 5.If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
		rf.commitIndex = min(args.LeaderCommit, len(rf.log)-1)
	}
	rf.mu.Unlock()
}

func (rf *Raft) handleAppendEntries(serverTo int, args *AppendEntriesArgs) {

	reply := &AppendEntriesReply{}
	ok := rf.sendAppendEntries(serverTo, args, reply)
	if !ok {
		return
	}

	rf.mu.Lock()

	if args.Term != rf.currentTerm {
		rf.mu.Unlock()
		return
	}

	if reply.Success {
		rf.matchIndex[serverTo] = args.PrevLogIndex + len(args.Entries)
		rf.nextIndex[serverTo] = rf.matchIndex[serverTo] + 1

		N := len(rf.log) - 1

		for N > rf.commitIndex {
			count := 1
			for i := 0; i < len(rf.peers); i++ {
				if i == rf.me {
					continue
				}
				if rf.matchIndex[i] >= N && rf.log[N].Term == rf.currentTerm {
					count += 1
				}
			}
			if count > len(rf.peers)/2 {
				rf.commitIndex = N
				break
			}
			N -= 1
		}

		rf.mu.Unlock()
		return
	}

	if reply.Term > rf.currentTerm {

		DPrintf("[%v] received an updated term: %v from the heartbeat function of the old leader from server %v, transitioning to Follower\n", rf.me, reply.Term, serverTo)
		rf.convertToFollower(reply.Term)
		rf.mu.Unlock()
		return
	}

	if reply.Term == rf.currentTerm && rf.state == Leader {
		rf.nextIndex[serverTo] -= 1
		rf.mu.Unlock()
		return
	}
	// }
}

func (rf *Raft) SendHeartBeats() {
	DPrintf("[%v] starts sending heartbeats\n", rf.me)

	for !rf.killed() {
		rf.mu.Lock()
		// if the server is dead or is not the leader, just return
		if rf.state != Leader {
			rf.mu.Unlock()
			// 不是leader则终止心跳的发送
			return
		}

		for i := 0; i < len(rf.peers); i++ {
			if i == rf.me {
				continue
			}
			args := &AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderID:     rf.me,
				PrevLogIndex: rf.nextIndex[i] - 1,
				PrevLogTerm:  rf.log[rf.nextIndex[i]-1].Term,
				LeaderCommit: rf.commitIndex,
			}
			if len(rf.log)-1 >= rf.nextIndex[i] {
				args.Entries = rf.log[rf.nextIndex[i]:]
				DPrintf("[%v] starts broadcasting new AppendEntries to server %v\n", rf.me, i)
			} else {
				args.Entries = make([]LogEntry, 0)
				DPrintf("[%v] starts broadcasting a new heartbeat to server %v, args = %+v \n", rf.me, i, args)
			}
			go rf.handleAppendEntries(i, args)
		}

		rf.mu.Unlock()

		time.Sleep(time.Duration(HeartBeatTimeOut) * time.Millisecond)
	}
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

func (rf *Raft) convertToCandidate() {
	rf.state = Candiate
	rf.currentTerm++
	rf.votedFor = rf.me
}

func (rf *Raft) convertToFollower(term int) {
	rf.state = Follower // peer is passive
	rf.currentTerm = term
	// rf.log = append(rf.log, LogEntry{Term:0})
	rf.votedFor = -1
	rf.lastReceive = time.Now()
}

func (rf *Raft) convertToLeader() {
	rf.state = Leader
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	lastIndex := rf.getLastLogIndex()
	for server := range rf.peers {
		rf.nextIndex[server] = lastIndex + 1
		rf.matchIndex[server] = 0
	}
	go rf.SendHeartBeats()

}

func (rf *Raft) getLastLogIndex() int {
	return len(rf.log) - 1 // 0-indexed slice
}
func (rf *Raft) getLastLogTerm() int {
	if rf.getLastLogIndex() < 0 {
		panic("empty slice")
	}
	return rf.log[rf.getLastLogIndex()].Term
}

func (rf *Raft) KickOffElection() {
	rf.mu.Lock()
	rf.convertToCandidate()
	term := rf.currentTerm
	args := RequestVoteArgs{
		Term:         term,
		CandidateId:  rf.me,
		LastLogIndex: rf.getLastLogIndex(),
		LastLogTerm:  rf.getLastLogTerm(),
	}
	DPrintf("[%v] Started Election at term: %v", rf.me, rf.currentTerm)
	rf.mu.Unlock()
	voteCh := make(chan bool, len(rf.peers))
	for server := range rf.peers {
		if server == rf.me {
			voteCh <- true
			continue
		}
		go func(server int) {
			reply := RequestVoteReply{}
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

	// collect votes and do the counting
	rf.countVotes(voteCh, term)
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
				DPrintf("[%d] convert to Leader", rf.me)
				rf.convertToLeader()
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

func getRandomElectTimeOut(rd *rand.Rand) int {
	plusMs := int(rd.Float64() * 150)

	return plusMs + ElectTimeOutBase
}

func (rf *Raft) ticker() {
	rd := rand.New(rand.NewSource(int64(rf.me)))
	for !rf.killed() {

		// Your code here (2A)
		// Check if a leader election should be started.

		rdTimeOut := getRandomElectTimeOut(rd)
		rf.mu.Lock()
		if rf.state != Leader && time.Since(rf.lastReceive) > time.Duration(rdTimeOut)*time.Millisecond {
			go rf.KickOffElection()
		}
		rf.mu.Unlock()
		time.Sleep(ElectTimeOutCheckInterval)
	}
}

func (rf *Raft) CommitChecker() {
	for !rf.killed() {
		rf.mu.Lock()
		for rf.commitIndex > rf.lastApplied {
			rf.lastApplied += 1
			msg := ApplyMsg{
				CommandValid: true,
				Command:      rf.log[rf.lastApplied].Command,
				CommandIndex: rf.lastApplied,
			}
			rf.applyCh <- msg
			DPrintf("[%v] is preparing to apply command %v (index %v) to the state machine\n", rf.me, msg.Command, msg.CommandIndex)
		}
		rf.mu.Unlock()
		time.Sleep(CommitCheckTimeInterval)
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
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.applyCh = applyCh
	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))
	rf.log = append(rf.log, LogEntry{Term: 0})
	rf.lastReceive = time.Now()

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	go rf.CommitChecker()

	return rf
}
