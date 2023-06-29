# 以下是一些规则：

- map阶段应将中间键分配到nReduce个reduce任务的桶中，其中nReduce是reduce任务的数量——这是main/mrcoordinator.go传递给MakeCoordinator()的参数。每个mapper应创建nReduce个中间文件供reduce任务使用。

- worker实现应将X个reduce任务的输出放在文件mr-out-X中。

- 每个mr-out-X文件应包含一行reduce函数的输出。这行应使用Go的"%v %v"格式生成，用键和值调用。你可以在main/mrsequential.go中看到注释为"this is the correct format"的行。如果你的实现与这种格式偏离太多，测试脚本会失败。

- 你可以修改mr/worker.go、mr/coordinator.go和mr/rpc.go。你可以临时修改其他文件进行测试，但要确保你的代码能与原始版本一起工作；我们将使用原始版本进行测试。

- worker应将中间Map输出放在当前目录的文件中，你的worker稍后可以将它们作为Reduce任务的输入读取。

- main/mrcoordinator.go期望mr/coordinator.go实现一个Done()方法，当MapReduce任务完全完成时返回true；此时，mrcoordinator.go将退出。

- 当任务完全完成时，worker进程应退出。实现这一点的一个简单方法是使用call()的返回值：如果worker无法联系到coordinator，它可以假定coordinator因为任务完成而退出，所以worker也可以终止。根据你的设计，你可能会发现coordinator能给worker一个“请退出”的伪任务也很有帮助。