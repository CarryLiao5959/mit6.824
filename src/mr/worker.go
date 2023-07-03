package mr

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"plugin"
)

const (
    Idle = iota
    InProgress
    Completed
)

type WorkerInfo struct {
	// add fields here
	Status int
	Id int
	File string
}


//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}


//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	
	// uncomment to send the Example RPC to the coordinator.
	//setJobDone(false)
	// CallExample()
	myInfo := WorkerInfo{}

	callRequest(&myInfo)

	fmt.Println(myInfo.File, myInfo.Id)

	//setJobDone(true)

}

func callRequest(myInfo *WorkerInfo) {

	// declare an argument structure.
	args := RequestArgs{}

	// fill in the argument(s).
	args.WorkerRequest = "this is a request"

	// declare a reply structure.
	reply := RequestReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.RequestHandler", &args, &reply)

	if ok {
		// reply.workerRequestReply
		fmt.Printf("[reply.RequestReply] %v\n", reply.WorkerRequestReply)
		myInfo.Status = InProgress
		myInfo.Id = reply.MyTask.TaskID
		myInfo.File = reply.MyTask.File
	} else {
		fmt.Printf("[call failed!]\n")
	}
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
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

func setJobDone(done bool) {

	args := JobDoneArgs{Status: done}
	reply := JobDoneReply{}

	ok := call("Coordinator.SetJobDone", &args, &reply)
	if ok {
		fmt.Println("Job done status set successfully")
	} else {
		fmt.Println("Failed to set job done status")
	}
}

func Map(filenames []string) {
	mapf := loadPluginMap(os.Args[1])

	intermediate := []KeyValue{}

	for _, filename := range filenames {
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
		intermediate = append(intermediate, kva...)
	}

}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)

	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// load the application Map and Reduce functions
// from a plugin file, e.g. ../mrapps/wc.so
// 接收一个字符串参数，表示插件文件的路径，
// 并返回两个函数，一个是 Map 函数，另一个是 Reduce 函数。
func loadPluginMap(filename string) (func(string, string) []KeyValue) {
	// 打开插件文件
	p, err := plugin.Open(filename)
	if err != nil {
		log.Fatalf("cannot load plugin %v", filename)
	}
	// 从插件中查找名为 "Map" 的函数
	xmapf, err := p.Lookup("Map")
	if err != nil {
		log.Fatalf("cannot find Map in %v", filename)
	}
	// 将查找到的 "Map" 函数强制转换为正确的类型，
	// 这里的类型是 func(string, string) []mr.KeyValue，也就是接收两个字符串参数，返回 mr.KeyValue 切片的函数。
	mapf := xmapf.(func(string, string) []KeyValue)

	return mapf
}