# 开始操作

你需要设置 Go 来完成实验。

用 git（一个版本控制系统）获取初始的实验软件。要了解更多关于 git 的信息，可以查看 Pro Git 书籍或 git 用户手册。

```bash
$ git clone git://g.csail.mit.edu/6.5840-golabs-2023 6.5840
$ cd 6.5840
$ ls
Makefile src
$
```

我们为你提供了一个简单的顺序 mapreduce 实现，在 src/main/mrsequential.go 中。它一次运行一次 maps 和 reduces，都在一个单独的进程中。我们还为你提供了一些 MapReduce 应用程序：mrapps/wc.go 中的 word-count，以及 mrapps/indexer.go 中的文本索引器。你可以如下运行顺序的 word count：

```bash
$ cd ~/6.5840
$ cd src/main
$ go build -buildmode=plugin ../mrapps/wc.go
$ rm mr-out*
$ go run mrsequential.go wc.so pg*.txt
$ more mr-out-0
A 509
ABOUT 2
ACT 8
...
```

mrsequential.go 将其输出保存在文件 mr-out-0 中。输入来自名为 pg-xxx.txt 的文本文件。

请随意从 mrsequential.go 中借用代码。你还应该看一下 mrapps/wc.go，看看 MapReduce 应用程序代码是什么样的。

对于这个实验和所有其他实验，我们可能会对我们提供的代码进行更新。为了确保你能够获取这些更新并使用 git pull 轻松地合并它们，最好将我们提供的代码保留在原始文件中。你可以按照实验说明书中的指示添加到我们提供的代码；只是不要移动它。你可以把你自己的新函数放在新文件中。