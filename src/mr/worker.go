package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	//somehow worker should be conscious of the nReduce val
	// Your worker implementation here.
	for {
		reply, succ := requestTask()
		if !succ {
			fmt.Println("Faile to get task from coordinator.")
			return
			//should i return or continue next loop to retry?
		}
		switch reply.TaskType {
		case MapTask:
			performMap(mapf, reply)
		case ReduceTask:
			performReduce()
		case ExitTask:
			fmt.Println("No task left. Worker shuting down.")
		case NoTask:
			fmt.Println("No task available. Waiting...")
			//avoid spin waiting
			time.Sleep(1 * time.Second)
		}
	}
	// uncomment to send the Example RPC to the coordinator.
	// CallExample()
}

// function to ask the coordinator for a new task.
func requestTask() (TaskResponse, bool) {
	args := TaskRequest{WorkerID: os.Getpid()}
	reply := TaskResponse{}

	ok := call("Coordinator.RequestTask", &args, &reply)
	return reply, ok
}

// function to execute map tasks
func performMap(mapf func(string, string) []KeyValue, task TaskResponse) {
	// 1. Read the input file
	filename := task.MapFile
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	//2.Apply user-defined mapf to get KV pairs
	kva := mapf(filename, string(content))
	// 3. Perform JSON encoding for intermediate files to handle arbitrary KV types
	intermediateFiles := make([]*os.File, task.NReduce)
	encoders := make([]*json.Encoder, task.NReduce)
	for i := 0; i < task.NReduce; i++ {
		intermediateFiles[i], err = createIntermediateFile(task.TaskID, i)
		if err != nil {
			log.Fatalf("cannot create intermediate file: %v", err)
		}
		encoders[i] = json.NewEncoder(intermediateFiles[i])
	}

	// Partition and encode key-value pairs
	for _, kv := range kva {
		reduceTaskNum := ihash(kv.Key) % task.NReduce
		err := encoders[reduceTaskNum].Encode(&kv)
		if err != nil {
			log.Fatalf("cannot encode key-value pair: %v", err)
		}
	}

	// Close intermediate files
	for _, f := range intermediateFiles {
		f.Close()
	}

	// 4. Notify task completion
	notifyTaskCompletion(task.TaskID, MapTask)
}

// Helper function to create intermediate files
func createIntermediateFile(mapTaskID, reduceTaskID int) (*os.File, error) {
	filename := fmt.Sprintf("mr-%d-%d", mapTaskID, reduceTaskID)
	return os.Create(filename)
}

// function to executre reduce tasks
func performReduce() {}

func notifyTaskCompletion(taskID int, taskType TaskType) {
	//this will use an RPC call to notify completion of a task
	args := TaskCompletion{
		WorkerID: os.Getpid(),
		TaskID:   taskID,
		TaskType: taskType,
	}
	reply := struct{}{}

	//permorm call
	call("Coordinator.NotifyTaskCompletion", &args, &reply)
}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
// func CallExample() {

// 	// declare an argument structure.
// 	args := ExampleArgs{}

// 	// fill in the argument(s).
// 	args.X = 99

// 	// declare a reply structure.
// 	reply := ExampleReply{}

// 	// send the RPC request, wait for the reply.
// 	// the "Coordinator.Example" tells the
// 	// receiving server that we'd like to call
// 	// the Example() method of struct Coordinator.
// 	ok := call("Coordinator.Example", &args, &reply)
// 	if ok {
// 		// reply.Y should be 100.
// 		fmt.Printf("reply.Y %v\n", reply.Y)
// 	} else {
// 		fmt.Printf("call failed!\n")
// 	}
// }

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
