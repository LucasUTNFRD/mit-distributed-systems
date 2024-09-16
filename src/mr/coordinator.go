package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TaskStatus int

const (
	Idle TaskStatus = iota
	InProgress
	Completed
)

type TaskType int

const (
	MapTask TaskType = iota
	ReduceTask
	ExitTask
	NoTask
)

type WorkerInfo struct {
	ID        int
	LastHeard time.Time
}

type Task struct {
	Type     TaskType
	Status   TaskStatus //faul tolerance considerations
	WorkerID int
	File     string
	TaskID   int
}

// TODO
// - Maintains the state of the job
// - Assigns tasks to Workers
// - Handles Worker failures
// - Coordinates the overall MapReduce process
type Phase int

const (
	MapPhase Phase = iota
	ReducePhase
	ExistPhase
)

type Coordinator struct {
	mu sync.Mutex

	mapTasks    []Task
	reduceTasks []Task
	nReduce     int
	nMap        int

	workers      map[int]*WorkerInfo
	nextWorkerID int

	intermediateFiles [][]string

	mapTasksCompleted    int
	reduceTasksCompleted int

	status Phase
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) RequestTask(args *TaskRequest, reply *TaskResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	workerPID := args.WorkerID

	// Update or create worker info
	if _, exists := c.workers[workerPID]; !exists {
		c.workers[workerPID] = &WorkerInfo{
			ID:        workerPID,
			LastHeard: time.Now(),
		}
	} else {
		c.workers[workerPID].LastHeard = time.Now()
	}
	// Assign task logic
	// Check and update phases
	if c.status == MapPhase && c.mapTasksCompleted == c.nMap {
		c.status = ReducePhase
		// Initialize reduce tasks if not already done
		if len(c.reduceTasks) == 0 {
			for i := 0; i < c.nReduce; i++ {
				c.reduceTasks = append(c.reduceTasks, Task{
					Type:   ReduceTask,
					Status: Idle,
					TaskID: i,
				})
			}
		}
	} else if c.status == ReducePhase && c.reduceTasksCompleted == c.nReduce {
		c.status = ExistPhase
	}

	// Assign tasks based on current phase
	switch c.status {
	case MapPhase:
		for i, task := range c.mapTasks {
			if task.Status == Idle {
				c.mapTasks[i].Status = InProgress
				c.mapTasks[i].WorkerID = args.WorkerID
				reply.TaskType = MapTask
				reply.TaskID = task.TaskID
				reply.MapFile = task.File
				reply.NReduce = c.nReduce
				go c.waitTask(&c.mapTasks[i])
				return nil
			}
		}

	case ReducePhase:
		for i, task := range c.reduceTasks {
			if task.Status == Idle {
				c.reduceTasks[i].Status = InProgress
				c.reduceTasks[i].WorkerID = args.WorkerID
				reply.TaskType = ReduceTask
				reply.TaskID = task.TaskID
				reply.ReduceFiles = c.intermediateFiles[task.TaskID]
				go c.waitTask(&c.reduceTasks[i])
				return nil
			}
		}

	case ExistPhase:
		reply.TaskType = ExitTask
		return nil
	}
	// If no tasks available, tell worker to wait
	reply.TaskType = NoTask
	return nil
}

//If we receive a message, saying that a task has been done, we need to do the following:
// Check the task type, state of the task, and whether the result comes back from someone who is actually responsible for the job.
// Change the task status
// If all tasks (map and reduce) have been finished, get back to the workers, telling them that we have done all the jobs.

func (c *Coordinator) NotifyTaskCompletion(args *TaskCompletion, reply *TaskResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	log.Printf("Received task request from worker %d", args.WorkerID)
	switch args.TaskType {
	case MapTask:
		if c.mapTasks[args.TaskID].Status == InProgress && c.mapTasks[args.TaskID].WorkerID == args.WorkerID {
			c.mapTasks[args.TaskID].Status = Completed
			c.mapTasksCompleted++
			for _, file := range args.OutputFiles {
				reduceTaskNum := extractReduceTaskNum(file)
				c.intermediateFiles[reduceTaskNum] = append(c.intermediateFiles[reduceTaskNum], file)
			}
		}
	case ReduceTask:
		if c.reduceTasks[args.TaskID].Status == InProgress && c.reduceTasks[args.TaskID].WorkerID == args.WorkerID {
			c.reduceTasks[args.TaskID].Status = Completed
			c.reduceTasksCompleted++
		}
	}
	return nil
}

func extractReduceTaskNum(filename string) int {
	parts := strings.Split(filename, "-")
	if len(parts) != 3 {
		log.Fatalf("unexpected intermediate filename format: %s", filename)
	}
	num, err := strconv.Atoi(parts[2])
	if err != nil {
		log.Fatalf("error parsing reduce task number from filename %s: %v", filename, err)
	}
	return num
}
func (c *Coordinator) waitTask(task *Task) {
	time.Sleep(10 * time.Second)
	c.mu.Lock()
	defer c.mu.Unlock()
	if task.Status == InProgress {
		task.Status = Idle
		task.WorkerID = 0
	}
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
// func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
// 	reply.Y = args.X + 1
// 	return nil
// }

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == ExistPhase {
		if c.mapTasksCompleted == c.nMap && c.reduceTasksCompleted == c.nReduce {
			ret = true
		}
	}

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	c.nReduce = nReduce
	c.nMap = len(files)
	c.mapTasks = make([]Task, c.nMap)
	c.reduceTasks = make([]Task, c.nReduce)
	for i, file := range files {
		c.mapTasks[i] = Task{Type: MapTask, Status: Idle, File: file, TaskID: i}
	}
	for i := 0; i < c.nReduce; i++ {
		c.reduceTasks[i] = Task{Type: ReduceTask, Status: Idle, File: "", TaskID: i}
	}
	c.intermediateFiles = make([][]string, nReduce)
	c.status = MapPhase
	c.workers = make(map[int]*WorkerInfo)

	os.Mkdir(TempDir, 0755)
	c.server()
	return &c
}
