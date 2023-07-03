package mr

import "log"
import "net"
import "os"
import "net/rpc"
import "net/http"
import "fmt"

type Task struct {
	File   string
	TaskID int
}

type Coordinator struct {
	// add fields here
	// tasks []Task // Task is a custom type representing a task to be done
	jobsDone      bool
	Files         []string
	MapTask       chan Task
	NumMapTask    int
	ReduceTask    chan Task
	NumReduceTask int
	// workers []WorkerStatus // WorkerStatus is a custom type representing the status of a worker
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) RequestHandler(args *RequestArgs, reply *RequestReply) error {
	reply.WorkerRequestReply = "this is a request reply"
	reply.MyTask = <-c.MapTask
	return nil
}

func (c *Coordinator) SetJobDone(args *JobDoneArgs, reply *JobDoneReply) error {
	c.jobsDone = args.Status
	reply.Success = true
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
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

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	// ret := false
	// ret := true

	// Your code here.

	return c.jobsDone

}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		jobsDone:      false,
		Files:         files,
		MapTask:       make(chan Task,len(files)),
		NumMapTask:    3,
		ReduceTask:    make(chan Task),
		NumReduceTask: nReduce,
	}

	go func() {
        for i, file := range files {
            c.MapTask <- Task{File:file, TaskID:i}
			fmt.Println(file,i)
        }
        //close(c.MapTask) 
    }()

	// Your code here.

	c.server()

	return &c
}
