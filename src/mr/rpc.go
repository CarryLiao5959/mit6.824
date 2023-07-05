package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"hash/fnv"
	"os"
	"strconv"
)

// TaskType
const (
	Idle = iota
	InProgress
	Completed
)

type Task struct {
	Type    int
	File    string
	ID      int
	NReduce int
}

// Worker Status
const (
	MapWork = iota
	ReduceWork
	Waiting
	Exit
)

type WorkerInfo struct {
	Status     int
	ID         int
	WorkerTask Task
}

type RequestTaskArgs struct {
	RequestWords string
}

type RequestTaskReply struct {
	WorkerTask Task
	ReplyWords string
}

type MapTaskDoneArgs struct {
	TaskDone bool
}

type MapTaskDoneReply struct {
	WorkerTask Task
}

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

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
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
