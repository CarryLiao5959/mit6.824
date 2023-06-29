# 无需学分的挑战练习

实现你自己的MapReduce应用（参考mrapps/*中的示例），例如，分布式Grep（参见MapReduce论文的第2.3节）。

让你的MapReduce协调器和工作节点在不同的机器上运行，就像实践中一样。你需要设置你的RPC以通过TCP/IP而不是Unix套接字进行通信（参见Coordinator.server()中被注释掉的行），并使用共享的文件系统进行文件的读写。例如，你可以ssh登录到MIT的多台Athena集群机器，它们使用AFS来共享文件；或者，你可以租用一些AWS实例，并使用S3进行存储。