package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"time"
)

const TempDir = "temp"

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
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("Worker %d starting task: %v", os.Getpid(), reply.TaskType)
		switch reply.TaskType {
		case MapTask:
			performMap(mapf, reply)
			reportTaskDone(MapTask, reply.NReduce, reply.TaskID)
		case ReduceTask:
			performReduce(reducef, reply)
			reportTaskDone(ReduceTask, reply.NReduce, reply.TaskID)
		case ExitTask:
			fmt.Println("No task left. Worker shuting down.")
			return
		case NoTask:
			fmt.Println("No task available. Waiting...")
			//avoid spin waiting
			time.Sleep(1 * time.Second)
		}
	}
	// uncomment to send the Example RPC to the coordinator.
	// CallExample()
}

func reportTaskDone(taskType TaskType, nReduce, taskID int) {
	args := TaskCompletion{
		WorkerID: os.Getpid(),
		TaskType: taskType,
		TaskID:   taskID,
	}

	// For Map tasks, we need to report the intermediate files
	if taskType == MapTask {
		args.OutputFiles = make([]string, 0)
		for i := 0; i < nReduce; i++ {
			filename := fmt.Sprintf("%v/mr-%d-%d", TempDir, taskID, i)
			args.OutputFiles = append(args.OutputFiles, filename)
		}
	}

	reply := TaskResponse{}
	ok := call("Coordinator.NotifyTaskCompletion", &args, &reply)
	if !ok {
		log.Printf("Failed to report task completion for %v task %d", taskType, taskID)
	}
}

// function to ask the coordinator for a new task.
func requestTask() (TaskResponse, bool) {
	args := TaskRequest{WorkerID: os.Getpid()}
	reply := TaskResponse{}

	ok := call("Coordinator.RequestTask", &args, &reply)
	return reply, ok
}

func performMap(mapf func(string, string) []KeyValue, task TaskResponse) {
	filename := task.MapFile
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	kva := mapf(filename, string(content))

	writeMapOutput(kva, task.TaskID, task.NReduce)
}

func writeMapOutput(kva []KeyValue, mapID int, nReduce int) {
	tempFiles := make([]*os.File, nReduce)
	encoders := make([]*json.Encoder, nReduce)

	for i := 0; i < nReduce; i++ {
		tempFile, err := ioutil.TempFile(TempDir, "intermediate-")
		if err != nil {
			log.Fatalf("cannot create temp file")
		}
		tempFiles[i] = tempFile
		encoders[i] = json.NewEncoder(tempFiles[i])
	}

	for _, kv := range kva {
		reduceID := ihash(kv.Key) % nReduce
		err := encoders[reduceID].Encode(&kv)
		if err != nil {
			log.Fatalf("cannot encode kv")
		}
	}

	for i, tempFile := range tempFiles {
		tempFile.Close()
		filename := fmt.Sprintf("%v/mr-%d-%d", TempDir, mapID, i)
		os.Rename(tempFile.Name(), filename)
	}
}

func performReduce(reducef func(string, []string) string, task TaskResponse) {
	files, err := filepath.Glob(fmt.Sprintf("%v/mr-*-%d", TempDir, task.TaskID))
	if err != nil {
		log.Fatalf("cannot read temp files")
	}

	intermediate := make(map[string][]string)

	for _, filepath := range files {
		file, err := os.Open(filepath)
		if err != nil {
			log.Fatalf("cannot open temp file")
		}

		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			err := dec.Decode(&kv)
			if err != nil {
				break
			}
			intermediate[kv.Key] = append(intermediate[kv.Key], kv.Value)
		}

		file.Close()
	}

	outputFile, err := ioutil.TempFile(TempDir, "mr-out-")
	if err != nil {
		log.Fatalf("cannot create temp file")
	}
	defer outputFile.Close()

	for key, values := range intermediate {
		output := reducef(key, values)
		fmt.Fprintf(outputFile, "%v %v\n", key, output)
	}

	finalFilename := fmt.Sprintf("mr-out-%d", task.TaskID)
	os.Rename(outputFile.Name(), finalFilename)
}

// func notifyTaskCompletion(taskID int, taskType TaskType, outputFiles []string) {
// 	//this will use an RPC call to notify completion of a task
// 	args := TaskCompletion{
// 		WorkerID:    os.Getpid(),
// 		TaskID:      taskID,
// 		TaskType:    taskType,
// 		OutputFiles: outputFiles,
// 	}
// 	reply := struct{}{}
// 	log.Printf("Received completion notification for %v task %d from worker %d", args.TaskType, args.TaskID, args.WorkerID)

// 	//perform call
// 	call("Coordinator.NotifyTaskCompletion", &args, &reply)
// }

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
