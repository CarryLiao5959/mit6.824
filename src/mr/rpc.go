package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

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
	MyTask Task
	WorkerRequestReply string
}

type JobDoneArgs struct {
	Status bool
}

type JobDoneReply struct {
	Success bool
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
