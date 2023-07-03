package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"plugin"
	"sort"
	"strconv"
	"time"
)

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
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {

	// Your worker implementation here.
	flag:=true

	for flag {
		// uncomment to send the Example RPC to the coordinator.
		//setJobDone(false)
		// CallExample()
		myInfo := WorkerInfo{}
		callRequest(&myInfo)
		switch myInfo.Type{
		case MapWork:
			{
				fmt.Println("[MapWork]")
				fmt.Println(myInfo.Status, myInfo.File, myInfo.Id)
				MapStore(myInfo)
				setJobDone(true)
			}
		case ReduceWork:
			{
				fmt.Println("[ReduceWork]")
				fmt.Println(myInfo.Status, myInfo.File, myInfo.Id)
			}
		case Waiting:
			{
				fmt.Println("[Wait]")
				fmt.Println(myInfo.Status, myInfo.File, myInfo.Id)
				time.Sleep(time.Second)
			}
		case Exit:
			{
				fmt.Println("[Exit]")
                fmt.Println(myInfo.Status, myInfo.File, myInfo.Id)
                flag=false
        	}
		}
		//setJobDone(true)
	}
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
		myInfo.nReduce = reply.NReduce
		fmt.Println("[reply.nReduce]",reply.NReduce)
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

	args := JobDoneArgs{TaskDone: done}
	reply := JobDoneReply{}

	ok := call("Coordinator.SetJobDone", &args, &reply)
	if ok {
		fmt.Println("Job done status set successfully")
	} else {
		fmt.Println("Failed to set job done status")
	}
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

func MapStore(myInfo WorkerInfo) {
	filename := myInfo.File
	mapf,reducef := loadPlugin(os.Args[1])
	//mapf := loadPluginMap(os.Args[1])

	intermediate := []KeyValue{}

	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	fmt.Println("[nReduce]",myInfo.nReduce)
	
	intermediate = mapf(filename, string(content))
	
	sort.Sort(ByKey(intermediate))
	
	// 对 intermediate 中的每一组相同的键进行 Reduce 操作
	i := 0
	reduced := []KeyValue{}
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		//fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)
		reduced = append(reduced, KeyValue{Key: intermediate[i].Key, Value: output})
		
		i = j
	}

	// 创建一个长度为nReduce的二维切片
	HashedKV := make([][]KeyValue, myInfo.nReduce)
	for _, kv := range reduced {
		HashedKV[ihash(kv.Key)%myInfo.nReduce] = append(HashedKV[ihash(kv.Key)%myInfo.nReduce], kv)
	}


	for i := 0; i < myInfo.nReduce; i++ {
		oname := "mr-tmp-" + strconv.Itoa(myInfo.Id) + "-" + strconv.Itoa(i)
		ofile, _ := os.Create("../../MapReduce/result/"+oname)
		fmt.Println("[Create success] MapReduce/result/",oname)
		enc := json.NewEncoder(ofile)
		for _, kv := range HashedKV[i] {
			enc.Encode(kv)
		}
		ofile.Close()
	}
	myInfo.Type = Waiting
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

func loadPlugin(filename string) (func(string, string) []KeyValue, func(string, []string) string) {
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
	
	// 查找和转换 "Reduce" 函数
	xreducef, err := p.Lookup("Reduce")
	if err != nil {
		log.Fatalf("cannot find Reduce in %v", filename)
	}
	reducef := xreducef.(func(string, []string) string)

	return mapf, reducef
}

func loadPluginMap(filename string) (func(string, string) []KeyValue) {
	p, err := plugin.Open(filename)
	if err != nil {
		log.Fatalf("cannot load plugin %v", filename)
	}
	xmapf, err := p.Lookup("Map")
	if err != nil {
		log.Fatalf("cannot find Map in %v", filename)
	}
	mapf := xmapf.(func(string, string) []KeyValue)
	return mapf
}