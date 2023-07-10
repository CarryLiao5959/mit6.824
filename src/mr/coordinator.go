package mr

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

var mu sync.Mutex

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
	TaskState     []Task
	TaskStateR    []Task
	Mutex         sync.Mutex
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
		TaskState:     make([]Task, len(files)),
		TaskStateR:    make([]Task, nReduce),
	}
	for i := range c.TaskState {
		c.TaskState[i].TaskStatus = Idle
	}
	for i := 0; i < c.NReduce; i++ {
		c.TaskStateR[i].TaskStatus = Idle
		task := Task{
			TaskStatus: Idle,
			File:       "mr-tmp-",
			ID:         i,
			NMap:       len(c.Files),
			NReduce:    c.NReduce,
		}
		c.ReduceTask <- task
		fmt.Println("[Create ReduceTask]", task.File, task.ID)
	}

	c.server()

	return &c
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false
	finished := true
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	for i, task := range c.TaskState {
		switch task.TaskStatus {
		case Idle:
			// task is ready
			finished = false
			c.addTask(i)
		case InProgress:
			// task is running
			finished = false
			c.checkTask(i)
		case Completed:
		default:
			panic("[Task State Error]")
		}
	}
	if finished {
		fmt.Println("[finished]")
		if c.Status == MapNotDone {
			fmt.Println("[c.Status == MapNotDone]")
			close(c.MapTask)
			c.Status = ReduceNotDone
		} else {
			fmt.Println("[AllDone]")
			c.Status = AllDone
			close(c.ReduceTask)
		}
	}
	ret = c.Status == AllDone
	return ret
}

// add Task to channel
func (c *Coordinator) addTask(taskIndex int) {
	c.TaskState[taskIndex].TaskStatus = Idle
	task := Task{
		TaskStatus: Idle,
		File:       "",
		ID:         taskIndex,
		NMap:       len(c.Files),
		NReduce:    c.NReduce,
	}
	if c.Status == MapNotDone {
		task.File = c.Files[taskIndex]
	}
	c.MapTask <- task
}

// check whether timeout
func (c *Coordinator) checkTask(taskIndex int) {
	timeDuration := time.Now().Sub(c.TaskState[taskIndex].StartTime)
	if timeDuration > MaxTaskRunTime {
		// if timeout: add to task channel back
		c.addTask(taskIndex)
	}
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
	fmt.Println("[RequestTaskHandler c.Status]", c.Status)
	if !args.WorkerAlive {
		return errors.New("[RequestTaskHandler] the Worker is offline")
	}
	if c.Status == MapNotDone {
		task, ok := <-c.MapTask
		reply.WorkerStatus = MapWork
		if ok {
			reply.ReplyWords = "[RequestMapTask Success]"
			c.TaskState[task.ID].TaskStatus = InProgress
			c.TaskState[task.ID].StartTime = time.Now()
			reply.WorkerTask = task
		} else {
			reply.ReplyWords = "[RequestMapTask Fail]"
			reply.WorkerTask = Task{NReduce: c.NReduce}
		}
	} else if c.Status == ReduceNotDone {
		task, ok := <-c.ReduceTask
		reply.WorkerStatus = ReduceWork
		if ok {
			reply.ReplyWords = "[RequestReduceTask Success]"
			c.TaskStateR[task.ID].TaskStatus = InProgress
			c.TaskStateR[task.ID].StartTime = time.Now()
			reply.WorkerTask = task
		} else {
			reply.ReplyWords = "[RequestReduceTask Fail]"
			reply.WorkerTask = Task{NReduce: c.NReduce}
		}
	} else if c.Status == AllDone {
		reply.ReplyWords = "[RequestReduceTask AllDone]"
		reply.WorkerStatus = AllDone
	}
	return nil
}

func (c *Coordinator) SetTaskDone(args *TaskDoneArgs, reply *TaskDoneReply) error {
	if !args.WorkerAlive {
		reply.ACK = false
		return errors.New("[SetTaskDone] the Worker is offline")
	}
	mu.Lock()
	defer mu.Unlock()
	if args.TaskDone {
		if c.Status == MapNotDone {
			c.TaskState[args.TaskID].TaskStatus = Completed
			c.NumMapDone++
			fmt.Println("[NumMapDone]", c.NumMapDone)
			if c.NumMapDone == len(c.Files) {
				c.Status = ReduceNotDone
			}
		} else if c.Status == ReduceNotDone {
			c.TaskStateR[args.TaskID].TaskStatus = Completed
			c.NumReduceDone++
			fmt.Println("[NumReduceDone]", c.NumReduceDone)
			if c.NumReduceDone == c.NReduce {
				fmt.Println("[OMG] c.NumReduceDone == c.NReduce")
				reply.IfExit = true
			}
			if c.NumReduceDone == c.NReduce+1 {
				c.Status = AllDone
			}
		}
	}
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply TaskStatuss are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}
