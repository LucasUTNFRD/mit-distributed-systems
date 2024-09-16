package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

// type ExampleArgs struct {
// 	X int
// }

// type ExampleReply struct {
// 	Y int
// }

// TaskRequest is used by workers to request a task
type TaskRequest struct {
	WorkerID int
}

// TaskResponse is used by the coordinator to assign a task to a worker
type TaskResponse struct {
	TaskType    TaskType
	TaskID      int
	MapFile     string   // For map tasks
	ReduceFiles []string // For reduce tasks
	NReduce     int      // Number of reduce tasks (needed for map tasks)
}

// TaskCompletion is used by workers to notify the coordinator of task completion
type TaskCompletion struct {
	WorkerID    int
	TaskType    TaskType
	TaskID      int
	OutputFiles []string // For map tasks to report intermediate files
}

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
