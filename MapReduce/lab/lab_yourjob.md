# 你的任务（中等/困难）

你的任务是实现一个分布式MapReduce，由两个程序组成，协调器和工作器。只有一个协调器进程，以及一个或多个并行执行的工作器进程。在真实的系统中，工作器会在一堆不同的机器上运行，但在这个实验中，你会在一台机器上运行它们所有。工作器将通过RPC与协调器通信。每个工作器进程将向协调器请求任务，从一个或多个文件中读取任务的输入，执行任务，并将任务的输出写入一个或多个文件。协调器应该注意到如果一个工作器在合理的时间内（对于这个实验，使用十秒）没有完成任务，就将同样的任务交给另一个工作器。

我们已经给你一些初始的代码。协调器和工作器的"main"程序在main/mrcoordinator.go和main/mrworker.go中；不要修改这些文件。你应该将你的实现放在mr/coordinator.go，mr/worker.go和mr/rpc.go中。

以下是如何在word-count MapReduce应用程序上运行你的代码。首先，确保word-count插件是新构建的：

```bash
$ go build -buildmode=plugin ../mrapps/wc.go
```

在主目录中，运行协调器。

```bash
$ rm mr-out*
$ go run mrcoordinator.go pg-*.txt
```

向mrcoordinator.go的pg-*.txt参数是输入文件；每个文件对应一个"split"，并且是一个Map任务的输入。

在一个或多个其他窗口中，运行一些工作器：

```bash
$ go run mrworker.go wc.so
```

当工作器和协调器完成时，查看mr-out-*中的输出。当你完成实验后，输出文件的排序并集应该与顺序输出匹配，如下所示：

```bash
$ cat mr-out-* | sort | more
A 509
ABOUT 2
ACT 8
...
```

我们为你提供了一个测试脚本main/test-mr.sh。这个测试检查当给定pg-xxx.txt文件作为输入时，wc和indexer MapReduce应用程序是否产生正确的输出。测试还检查你的实现是否并行运行Map和Reduce任务，以及你的实现是否能够从运行任务时崩溃的工作器中恢复。

如果你现在运行测试脚本，它将挂起，因为协调器永远不会完成：

```bash
$ cd ~/6.5840/src/main
$ bash test-mr.sh
*** Starting wc test.
```

你可以在mr/coordinator.go中的Done函数中将ret := false更改为true，以便协调器立即退出。然后：

```bash
$ bash test-mr.sh
*** Starting wc test.
sort: No such file or directory
cmp: EOF on mr-wc-all
--- wc output is not the same as mr
-correct-wc.txt
--- wc test: FAIL
$
```

测试脚本期望在以mr-out-X命名的文件中看到输出，每个reduce任务一个。mr/coordinator.go和mr/worker.go的空实现不会生成这些文件（或者做任何其他的事情），所以测试失败。

当你完成时，测试脚本的输出应该如下所示：

```bash
$ bash test-mr.sh
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting job count test.
--- job count test: PASS
*** Starting early exit test.
--- early exit test: PASS
*** Starting crash test.
--- crash test: PASS
*** PASSED ALL TESTS
$
```

你可能会看到来自Go RPC包的一些错误，看起来像这样

```bash
2019/12/16 13:27:09 rpc.Register: method "Done" has 1 input parameters; needs exactly three
```

忽略这些消息；注册协调器作为RPC服务器检查所有的方法是否适合RPC（有3个输入）；我们知道Done不是通过RPC调用的。