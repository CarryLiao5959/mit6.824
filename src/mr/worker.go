package mr

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	flag := true

	for flag {
		reply := RequestTaskReply{}
		reply = callRequestTask()
		fmt.Println("[nReduce]", reply.WorkerTask.NReduce)
	LOOP:
		switch reply.WorkerStatus {
		case MapWork:
			{
				fmt.Println("[MapWork]")
				MapAndStore(mapf, reducef, reply.WorkerTask)
				setTaskDone(true, reply.WorkerTask.ID)
			}
		case ReduceWork:
			{
				fmt.Println("[ReduceWork]")
				ReduceAndStore(reducef, reply.WorkerTask)
				taskDoneReply := setTaskDone(true, reply.WorkerTask.ID)
				if taskDoneReply.IfExit {
					reply.WorkerStatus = Exit
					goto LOOP
				}
			}
		case Waiting:
			{
				fmt.Println("[Waiting]")
				time.Sleep(2 * time.Second)
			}
		case Exit:
			{
				fmt.Println("[Exit]")
				setTaskDone(true, reply.WorkerTask.ID)
				flag = false
			}
		}
	}
}

func callRequestTask() RequestTaskReply {
	fmt.Println("[callRequestTask]")
	args := RequestTaskArgs{WorkerAlive: true, RequestWords: "[RequestWords]"}
	reply := RequestTaskReply{}
	ok := call("Coordinator.RequestTaskHandler", &args, &reply)
	if !ok {
		log.Fatal("[callRequestTask Fail]\n")
	}
	fmt.Println("[reply.ReplyWords]", reply.ReplyWords)
	fmt.Println("[callRequestTask nReduce]", reply.WorkerTask.NReduce)
	fmt.Println("[WorkerTask.File]", reply.WorkerTask.File)
	return reply
}

func setTaskDone(done bool, taskIndex int) TaskDoneReply {

	args := TaskDoneArgs{WorkerAlive: true, TaskDone: done, TaskID: taskIndex}
	reply := TaskDoneReply{}

	ok := call("Coordinator.SetTaskDone", &args, &reply)
	if !ok {
		log.Fatal("[SetTaskDone Fail]")
	}
	return reply
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

func MapAndStore(mapf func(string, string) []KeyValue, reducef func(string, []string) string, task Task) {
	nReduce := task.NReduce
	filename := task.File
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()

	// Map
	maped := mapf(filename, string(content))
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

	// Store
	for i := 0; i < nReduce; i++ {
		oname := "mr-tmp-" + strconv.Itoa(task.ID) + "-" + strconv.Itoa(i)
		ofile, _ := os.Create(oname)
		// ofile, _ := os.Create("../../MapReduce/result/" + oname)
		// ofile, _ := os.Create("./mr-tmp/" + oname)
		fmt.Println("[Store Success]", oname)
		enc := json.NewEncoder(ofile)
		for _, kv := range reduced {
			// -> nReduce
			if ihash(kv.Key)%nReduce == i {
				enc.Encode(&kv)
			}
		}
		ofile.Close()
	}
}

func ReduceAndStore(reducef func(string, []string) string, task Task) {
	// nReduce := myWorkerInfo.WorkerTask.NReduce
	nMap := task.NMap
	id := task.ID

	filenameprefix := task.File

	intermediate := []KeyValue{}

	for i := 0; i < nMap; i++ {
		filename := filenameprefix + strconv.Itoa(i) + "-" + strconv.Itoa(id)
		// filename := "../../MapReduce/result/" + filenameprefix + strconv.Itoa(i) + "-" + strconv.Itoa(id)
		// filename := "./mr-tmp/" + filenameprefix + strconv.Itoa(i) + "-" + strconv.Itoa(id)
		fmt.Println("[Reduce Open]", filename)
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
		file.Close()
	}

	sort.Sort(ByKey(intermediate))

	// Reduce
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

		sum := 0
	OuterLoop:
		for _, v := range values {
			for _, runeValue := range v {
				if runeValue < '0' || runeValue > '9' {
					break OuterLoop
				}
			}
			val, err := strconv.Atoi(v)
			if err != nil {
				log.Fatalf("[ReduceAndStore] Cannot convert string to integer: %v", err)
			}
			sum += val
		}
		reduced = append(reduced, KeyValue{Key: intermediate[i].Key, Value: strconv.Itoa(sum)})

		i = j
	}

	// Store
	index := task.ID
	oname := "mr-out-" + strconv.Itoa(index)
	ofile, err := os.Create(oname)
	// ofile, err := os.Create("../../MapReduce/result/" + oname)
	// ofile, err := os.Create("./mr-tmp/" + oname)
	if err != nil {
		log.Fatalf("cannot create %v", oname)
	}
	defer ofile.Close()
	fmt.Println("[Store Success]", oname)

	for _, kv := range reduced {
		_, err := fmt.Fprintf(ofile, "%v %v\n", kv.Key, kv.Value)
		if err != nil {
			log.Fatalf("cannot write to %v", oname)
		}
	}
}

func MergeFile(nReduce int) {
	intermediate := []KeyValue{}

	for i := 0; i < nReduce; i++ {
		filename := "mr-out-" + strconv.Itoa(i)
		// filename := "../../MapReduce/result/mr-out-" + strconv.Itoa(i)
		// filename := "./mr-tmp/mr-out-" + strconv.Itoa(i)
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				log.Fatalf("invalid line in %v: %v", filename, line)
			}
			intermediate = append(intermediate, KeyValue{Key: parts[0], Value: parts[1]})
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("cannot read %v: %v", filename, err)
		}
		file.Close()
	}

	sort.Sort(ByKey(intermediate))

	oname := "mr-wc-all"
	// oname := "../../MapReduce/result/mr-wc-all"
	// oname := "./mr-tmp/mr-wc-all"
	ofile, err := os.Create(oname)
	if err != nil {
		log.Fatalf("cannot create %v: %v", oname, err)
	}
	for _, kv := range intermediate {
		fmt.Fprintf(ofile, "%v %v\n", kv.Key, kv.Value)
	}
	ofile.Close()
}
