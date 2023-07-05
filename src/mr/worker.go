package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"plugin"
	"sort"
	"strconv"
	"time"
)

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	flag := true

	for flag {
		myWorkerInfo := WorkerInfo{}
		callRequestTask(&myWorkerInfo)

		switch myWorkerInfo.Status {
		case MapWork:
			{
				fmt.Println("[MapWork]")
				fmt.Println(myWorkerInfo.Status, myWorkerInfo.ID)
				MapAndStore(myWorkerInfo)
				setMapTaskDone(true)
			}
		case ReduceWork:
			{
				fmt.Println("[ReduceWork]")
				fmt.Println(myWorkerInfo.Status, myWorkerInfo.ID)
			}
		case Waiting:
			{
				fmt.Println("[Waiting]")
				fmt.Println(myWorkerInfo.Status, myWorkerInfo.ID)
				time.Sleep(10 * time.Second)
			}
		case Exit:
			{
				fmt.Println("[Exit]")
				fmt.Println(myWorkerInfo.Status, myWorkerInfo.ID)
				flag = false
			}
		}
	}
}

func callRequestTask(myWorkerInfo *WorkerInfo) {
	args := RequestTaskArgs{}
	args.RequestWords = "[RequestWords]"
	reply := RequestTaskReply{}
	ok := call("Coordinator.RequestTaskHandler", &args, &reply)
	if ok {
		fmt.Printf(reply.ReplyWords)
		if reply.WorkerTask.File == "" {
			myWorkerInfo.Status = Waiting
		} else {
			myWorkerInfo.Status = MapWork
			myWorkerInfo.ID = 0
			myWorkerInfo.WorkerTask = reply.WorkerTask
		}
	} else {
		fmt.Printf("[callRequestTask Fail]\n")
	}
}

func setMapTaskDone(done bool) {

	args := MapTaskDoneArgs{TaskDone: done}
	reply := MapTaskDoneReply{}

	ok := call("Coordinator.SetMapTaskDone", &args, &reply)
	if ok {
		fmt.Println("[SetTaskDone Success]")
	} else {
		fmt.Println("[SetTaskDone Fail]")
	}
}

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

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

func MapAndStore(myWorkerInfo WorkerInfo) {
	nReduce := myWorkerInfo.WorkerTask.NReduce

	filename := myWorkerInfo.WorkerTask.File
	mapf, reducef := loadPlugin(os.Args[1])
	
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	fmt.Println("[nReduce]", nReduce)
	
	// Map
	maped := []KeyValue{}
	maped = mapf(filename, string(content))

	sort.Sort(ByKey(maped))

	// Reduce
	i := 0
	reduced := []KeyValue{}
	for i < len(maped) {
		j := i + 1
		for j < len(maped) && maped[j].Key == maped[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, maped[k].Value)
		}
		output := reducef(maped[i].Key, values)

		reduced = append(reduced, KeyValue{Key: maped[i].Key, Value: output})
		i = j
	}

	// -> nReduce
	HashedKV := make([][]KeyValue, nReduce)
	for _, kv := range reduced {
		HashedKV[ihash(kv.Key)%nReduce] = append(HashedKV[ihash(kv.Key)%nReduce], kv)
	}

	// Store
	for i := 0; i < nReduce; i++ {
		oname := "mr-tmp-" + strconv.Itoa(myWorkerInfo.WorkerTask.ID) + "-" + strconv.Itoa(i)
		ofile, _ := os.Create("../../MapReduce/result/" + oname)
		fmt.Println("[Store Success]", oname)
		enc := json.NewEncoder(ofile)
		for _, kv := range HashedKV[i] {
			enc.Encode(kv)
		}
		ofile.Close()
	}

	myWorkerInfo.WorkerTask.Type = Completed
}


func loadPlugin(filename string) (func(string, string) []KeyValue, func(string, []string) string) {
	
	p, err := plugin.Open(filename)
	if err != nil {
		log.Fatalf("cannot load plugin %v", filename)
	}
	
	xmapf, err := p.Lookup("Map")
	if err != nil {
		log.Fatalf("cannot find Map in %v", filename)
	}

	mapf := xmapf.(func(string, string) []KeyValue)

	xreducef, err := p.Lookup("Reduce")
	if err != nil {
		log.Fatalf("cannot find Reduce in %v", filename)
	}
	reducef := xreducef.(func(string, []string) string)

	return mapf, reducef
}

func loadPluginMap(filename string) func(string, string) []KeyValue {
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

func loadPluginReduce(filename string) (func(string, []string) string) {
	p, err := plugin.Open(filename)
	if err != nil {
		log.Fatalf("cannot load plugin %v", filename)
	}

	xreducef, err := p.Lookup("Reduce")
	if err != nil {
		log.Fatalf("cannot find Reduce in %v", filename)
	}
	reducef := xreducef.(func(string, []string) string)

	return reducef
}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
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

