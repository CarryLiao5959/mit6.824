package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

// Coordinator Status
const (
	MapNotDone = iota
	ReduceNotDone
	AllDone
)

type Coordinator struct {
	Status        int
	Files         []string
	MapTask       chan Task
	NumMapDone    int
	ReduceTask    chan Task
	NumReduceDone int
	NReduce       int
	Workers       chan WorkerInfo
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		Status:        MapNotDone,
		Files:         files,
		MapTask:       make(chan Task, 10),
		NumMapDone:    0,
		ReduceTask:    make(chan Task, nReduce),
		NumReduceDone: 0,
		NReduce:       nReduce,
		Workers:       make(chan WorkerInfo, 10),
	}

	fmt.Println("[FileLen]", len(files))

	go func() {
		defer close(c.MapTask)
		for i, file := range files {
			c.MapTask <- Task{
				Type:    Idle,
				File:    file,
				ID:      i,
				NReduce: c.NReduce,
			}
			fmt.Println(file, i)
		}
	}()

	if c.NumMapDone == len(files) {
		c.Status = ReduceNotDone
	}

	if true {
		c.Status = AllDone
	}

	c.server()

	return &c
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	return c.Status == ReduceNotDone
}

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

func (c *Coordinator) RequestTaskHandler(args *RequestTaskArgs, reply *RequestTaskReply) error {
	reply.ReplyWords = "[RequestTaskHandler]"
	select {
	case data, ok := <-c.MapTask:
		if !ok {
			reply.ReplyWords = "[RequestTask Fail]"
			reply.WorkerTask = Task{}
		} else {
			reply.ReplyWords = "[RequestTask Success]"
			reply.WorkerTask = data
		}
		fmt.Println(data.File)
		return nil
	}
}

func (c *Coordinator) SetMapTaskDone(args *MapTaskDoneArgs, reply *MapTaskDoneReply) error {
	if args.TaskDone {
		c.NumMapDone++
		fmt.Println("[NumJobsDone]", c.NumMapDone)
		if c.NumMapDone == len(c.Files) {
			c.Status = ReduceNotDone
		}
		reply.WorkerTask.Type = Completed
	}
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}
