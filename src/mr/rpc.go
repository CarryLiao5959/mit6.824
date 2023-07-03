package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

// Task
const (
	Idle = iota
	InProgress
	Completed
)

type Task struct {
	Type       int
	File       string
	TaskID     int
	TaskStatus int
}

// Worker
const (
	MapWork = iota
	ReduceWork
	Waiting
	Exit
)

type WorkerInfo struct {
	// add fields here
	Type    int
	Status  int
	Id      int
	File    string
	nReduce int
}

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

type RequestArgs struct {
	WorkerRequest string
}

type RequestReply struct {
	MyTask             Task
	NReduce            int
	WorkerRequestReply string
}

type JobDoneArgs struct {
	TaskDone bool
}

type JobDoneReply struct {
	TaskSuccess Task
}

// // example RequestTask RPC
// func (c *Coordinator) RequestTask(args *RequestTaskArgs, reply *RequestTaskReply) error {
// 	// implementation goes here
// }

// // example SubmitTask RPC
// func (c *Coordinator) SubmitTask(args *SubmitTaskArgs, reply *SubmitTaskReply) error {
// 	// implementation goes here
// }

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
