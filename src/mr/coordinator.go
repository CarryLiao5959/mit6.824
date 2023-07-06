package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
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

	c.server()

	for c.NumMapDone < len(files) {
	}
	close(c.MapTask)

	c.Status = ReduceNotDone

	go func() {
		for i := 0; i < c.NReduce; i++ {
			c.ReduceTask <- Task{
				Type:    Idle,
				File:    "mr-tmp-",
				ID:      i,
				NMap:    len(files),
				NReduce: c.NReduce,
			}
			fmt.Println("mr-tmp-" + strconv.Itoa(i))
		}
	}()

	fmt.Println("[nReduce]", nReduce)

	for c.NumReduceDone < nReduce {}

	close(c.ReduceTask)

	c.Status = AllDone

	return &c
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	return c.Status == AllDone
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
	if c.Status == MapNotDone {
		select {
		case data, ok := <-c.MapTask:
			if !ok {
				reply.ReplyWords = "[RequestMapTask Fail]"
				reply.WorkerInfoReply.Status = Waiting
				reply.WorkerInfoReply.WorkerTask = Task{}
			} else {
				reply.ReplyWords = "[RequestMapTask Success]"
				reply.WorkerInfoReply.Status = MapWork
				reply.WorkerInfoReply.WorkerTask = data
			}
			fmt.Println("[data.File]", data.File)
		}
	} else if c.Status == ReduceNotDone {
		select {
		case data, ok := <-c.ReduceTask:
			if !ok {
				reply.ReplyWords = "[RequestReduceTask Fail]"
				reply.WorkerInfoReply.Status = Waiting
				reply.WorkerInfoReply.WorkerTask = Task{}
			} else {
				reply.ReplyWords = "[RequestReduceTask Success]"
				reply.WorkerInfoReply.Status = ReduceWork
				reply.WorkerInfoReply.WorkerTask = data
			}
			fmt.Println("[data.File]", data.File)
		}
	} else if c.Status == AllDone {
		reply.ReplyWords = "[RequestReduceTask AllDone]"
		reply.WorkerInfoReply.Status = AllDone
	}
	return nil
}

func (c *Coordinator) SetTaskDone(args *TaskDoneArgs, reply *TaskDoneReply) error {
	if args.TaskDone {
		if c.Status == MapNotDone {
			c.NumMapDone++
			fmt.Println("[NumMapDone]", c.NumMapDone)
			if c.NumMapDone == len(c.Files) {
				reply.TaskWorker.Status = ReduceWork
				c.Status = ReduceNotDone
			}
			reply.TaskWorker.WorkerTask.Type = Completed
		} else if c.Status == ReduceNotDone {
			c.NumReduceDone++
			fmt.Println("[NumReduceDone]", c.NumReduceDone)
			if c.NumReduceDone == c.NReduce {
				fmt.Println("[OMG] c.NumReduceDone == c.NReduce")
				reply.TaskWorker.Status = Exit
				c.Status = AllDone
			}
			reply.TaskWorker.WorkerTask.Type = Completed
		}
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
