# 3 实现

许多不同的MapReduce接口实现是可能的。正确的选择取决于环境。例如，一个实现可能适合小型共享内存机器，另一个适合大型NUMA多处理器，而另一个则适合更大的网络机器集合。

这一节描述了一个针对在Google广泛使用的计算环境的实现：通过交换式以太网连接在一起的大型商品PC集群[4]。在我们的环境中：

（1）机器通常是双处理器的x86处理器，运行Linux，每台机器有2-4 GB的内存。

（2）使用商品网络硬件 - 通常在机器级别是100兆比特/秒或1千兆比特/秒，但总的切割带宽平均要低得多。

（3）一个集群由数百或数千台机器组成，因此机器故障是常见的。

（4）存储是由便宜的IDE磁盘提供的，这些磁盘直接连接到个别的机器上。一个由公司内部开发的分布式文件系统[8]用于管理存储在这些磁盘上的数据。文件系统使用复制在不可靠的硬件上提供可用性和可靠性。

（5）用户将任务提交给调度系统。每个任务由一组任务组成，并由调度器映射到集群内一组可用的机器上。

## 3.1 执行概述

通过自动将输入数据分割成M个分片，将Map调用分布在多个机器上。不同的机器可以并行处理输入分片。通过使用分区函数（例如，hash(key) mod R）将中间键空间分割成R个部分，将Reduce调用分布开。分区的数量（R）和分区函数由用户指定。

图1显示了我们的实现中一个MapReduce操作的整体流程。当用户程序调用MapReduce函数时，会发生以下一系列动作（图1中的编号标签对应于下面列表中的数字）：

1. 用户程序中的MapReduce库首先将输入文件分割成M个部分，每部分通常为16兆字节到64兆字节（MB）（用户可以通过一个可选参数控制）。然后它在机器群上启动程序的许多副本。

2. 程序的一个副本是特殊的 - 主节点。其余的都是工作节点，由主节点分配工作。有M个map任务和R个reduce任务可以分配。主节点选择空闲的工作节点，并为每一个分配一个map任务或一个reduce任务。

3. 分配map任务的工作节点读取相应输入分片的内容。它从输入数据中解析出键/值对，并将每对传递给用户定义的Map函数。由Map函数产生的中间键/值对在内存中缓冲。

4. 定期地，缓冲的对被写入本地磁盘，通过分区函数分割成R个区域。这些缓冲对在本地磁盘上的位置被传回给主节点，主节点负责将这些位置转发给reduce工作节点。

5. 当reduce工作节点被主节点通知这些位置时，它使用远程过程调用从map工作节点的本地磁盘读取缓冲数据。当reduce工作节点读取了所有中间数据后，它按照中间键对其进行排序，使得所有相同键的出现都被分组在一起。排序是必需的，因为通常许多不同的键映射到同一个reduce任务。如果中间数据量太大，无法装入内存，就使用外部排序。

6. reduce工作节点遍历排序后的中间数据，对于遇到的每个唯一的中间键，它将键和相应的中间值集合传递给用户的Reduce函数。Reduce函数的输出被附加到这个reduce分区的最终输出文件中。

7. 当所有的map任务和reduce任务都完成时，主节点唤醒用户程序。此时，用户程序中的MapReduce调用返回给用户代码。

成功完成后，mapreduce执行的输出在R个输出文件中可用（每个reduce任务一个，文件名由用户指定）。通常，用户不需要将这些R个输出文件合并成一个文件 - 他们通常将这些文件作为另一个MapReduce调用的输入，或者由能处理分成多个文件的输入的其他分布式应用程序使用。

## 3.2 主节点数据结构

主节点保持几个数据结构。对于每个map任务和reduce任务，它存储状态（空闲、进行中或已完成），以及工作机器的标识（对于非空闲任务）。

主节点是从map任务传递到reduce任务的中间文件区域位置的管道。因此，对于每个完成的map任务，主节点存储map任务产生的R个中间文件区域的位置和大小。随着map任务的完成，会收到这个位置和大小信息的更新。这些信息逐渐推送给有进行中的reduce任务的工作节点。

## 3.3 容错性

由于MapReduce库设计用于处理大量数据，涉及数百或数千台机器，所以该库必须能够优雅地处理机器故障。

### 工作节点故障：

主节点定期对每个工作节点进行ping操作。如果在一定的时间内没有从工作节点收到响应，主节点会将该工作节点标记为失败。该工作节点完成的所有map任务都会被重置为初始的空闲状态，因此可以在其他工作节点上进行调度。同样，任何在故障工作节点上进行的map任务或reduce任务也会被重置为空闲状态，变得可以重新调度。

因为完成的map任务的输出存储在故障机器的本地硬盘上，所以无法访问，所以在故障发生时需要重新执行这些任务。而已完成的reduce任务不需要重新执行，因为它们的输出存储在全局文件系统中。

当一个map任务首先由工作节点A执行，然后由工作节点B执行（因为A失败了），所有执行reduce任务的工作节点都会收到重新执行的通知。任何还没有从工作节点A读取数据的reduce任务将会从工作节点B读取数据。

MapReduce可以抵御大规模的工作节点故障。例如，在一次MapReduce操作中，正在运行的集群进行网络维护时，一次有80台机器在几分钟内无法访问。MapReduce主节点简单地重新执行无法访问的工作节点的工作，并继续前进，最终完成MapReduce操作。

### 主节点故障：

让主节点定期写入上述主数据结构的检查点是很容易的。如果主任务挂掉了，可以从最后一个检查点状态启动新的副本。但是，鉴于只有一个主节点，所以它的失败是不太可能的；因此，我们当前的实现在主节点失败时会中止MapReduce计算。客户端可以检查这种情况，并在需要时重试MapReduce操作。

### 故障存在时的语义：

当用户提供的map和reduce操作符是其输入值的确定性函数时，我们的分布式实现产生的输出与非故障顺序执行的整个程序产生的输出相同。

我们依赖map和reduce任务输出的原子提交来实现这个特性。每个正在进行的任务都将其输出写入私有临时文件。一个reduce任务产生一个这样的文件，一个map任务产生R个这样的文件（每个reduce任务一个）。当map任务完成时，工作节点向主节点发送一条消息，并在消息中包含R个临时文件的名称。如果主节点接收到一个已经完成的map任务的完成消息，它会忽略这个消息。否则，它会在主数据结构中记录R个文件的名字。

当reduce任务完成时，reduce工作节点将其临时输出文件原子性地重命名为最终输出文件。如果同一reduce任务在多台机器上执行，那么同一个最终输出文件会执行多次重命名操作。我们依赖底层文件系统提供的原子性重命名操作来保证最终文件系统状态只包含由reduce任务执行一次产生的数据。

我们的大多数map和reduce操作符都是确定性的，而且在这种情况下，我们的语义等价于顺序执行，这使得程序员很容易推理他们的程序的行为。当map和/或reduce操作符是非确定性的时，我们提供较弱但仍然合理的语义。在存在非确定性操作符的情况下，特定reduce任务R1的输出等同于由非确定性程序的顺序执行产生的R1的输出。然而，不同的reduce任务R2的输出可能对应于由非确定性程序的不同顺序执行产生的R2的输出。

考虑map任务M和reduce任务R1和R2。设e(Ri)为Ri的执行结果（有且只有一个这样的执行）。这种较弱的语义是因为e(R1)可能读取了M的一次执行产生的输出，而e(R2)可能读取了M的另一次执行产生的输出。

## 3.4 本地性

在我们的计算环境中，网络带宽是相对稀缺的资源。我们通过利用输入数据（由GFS[8]管理）存储在构成我们集群的机器的本地硬盘上的事实，来节省网络带宽。GFS将每个文件分为64MB的块，并在不同的机器上存储每个块的几个副本（通常是3个副本）。MapReduce主节点考虑到输入文件的位置信息，试图在包含相应输入数据副本的机器上调度map任务。如果失败，它会试图在该任务的输入数据副本附近调度map任务（例如，在包含数据的机器的同一网络交换机上的工作机器）。当在集群中的大部分工作节点上运行大型MapReduce操作时，大部分输入数据都是在本地读取的，不会消耗网络带宽。

## 3.5 任务粒度

如上所述，我们将map阶段划分为M个片段，reduce阶段划分为R个片段。理想情况下，M和R应该比工作节点的数量大得多。让每个工作节点执行许多不同的任务可以提高动态负载均衡，并且当一个工作节点失败时也可以加速恢复：它完成的许多map任务可以分布到所有其他的工作节点上。

在我们的实现中，M和R的大小有实际的限制，因为主节点必须做出O(M + R)个调度决策，并在内存中保持O(M * R)个状态，如上所述。（然而，内存使用的常数因子很小：O(M *R)状态的部分大约每个map任务/reduce任务对只包含一字节的数据）。此外，R通常受到用户的限制，因为每个reduce任务的输出都会在一个单独的输出文件中。实际上，我们通常选择M，使得每个单独的任务大约是16MB到64MB的输入数据（使得上述的本地化优化最有效），并且我们使R是我们预期使用的工作节点数量的小倍数。我们通常使用2000个工作节点进行MapReduce计算，M = 200,000，R = 5,000。

## 3.6 备份任务

导致MapReduce操作总时间延长的常见原因之一是“落后者”：一台机器在计算的最后几个map或reduce任务中花费异常长的时间。落后者可能由很多原因产生。例如，一台有坏硬盘的机器可能经常出现可纠正的错误，导致其读取性能从30MB/s下降到1MB/s。集群调度系统可能在该机器上调度了其他任务，由于CPU、内存、本地磁盘或网络带宽的竞争，导致它执行MapReduce代码的速度变慢。我们最近遇到的一个问题是机器初始化代码的一个bug，导致处理器缓存被禁用：受影响的机器上的计算速度下降了超过一百倍。

我们有一个通用的机制来缓解落后者的问题。当一个MapReduce操作接近完成时，主节点为剩余的进行中的任务调度备份执行。无论主执行还是备份执行何时完成，任务都会被标记为已完成。我们已经调整了这个机制，以使其通常增加的计算资源不超过总操作的几个百分点。我们发现，这大大减少了完成大型MapReduce操作的时间。例如，如第5.3节所述的排序程序，当备份任务机制被禁用时，完成时间增加了44%。