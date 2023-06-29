# 2 编程模型

计算接收一组输入键/值对，并产生一组输出键/值对。MapReduce库的用户通过两个函数来表达计算：Map和Reduce。

用户编写的Map函数接收一个输入对，并产生一组中间键/值对。MapReduce库将所有与同一中间键I关联的中间值组合在一起，并将它们传递给Reduce函数。

同样由用户编写的Reduce函数接受一个中间键I和一组该键的值。它将这些值合并在一起，形成可能较小的一组值。通常每次调用Reduce只产生零个或一个输出值。中间值通过迭代器提供给用户的reduce函数。这使我们能够处理过大以至于无法放入内存的值列表。

## 2.1 示例

考虑在大量文档集合中计算每个单词出现次数的问题。用户会编写类似于以下伪代码的代码：

```
map(String key, String value):
// key: 文档名
// value: 文档内容
for each word w in value:
EmitIntermediate(w, "1");

reduce(String key, Iterator values):
// key: 一个单词
// values: 一个计数列表
int result = 0;
for each v in values:
result += ParseInt(v);
Emit(AsString(result));
```

map函数发出每个单词以及关联的出现次数（在这个简单的例子中，只是‘1’）。reduce函数将为特定单词发出的所有计数相加。

此外，用户编写代码以填充mapreduce规范对象，其中包含输入和输出文件的名称，以及可选的调优参数。然后用户调用MapReduce函数，将规范对象传递给它。用户的代码与MapReduce库（用C++实现）一起链接。附录A包含此示例的完整程序文本。

## 2.2 类型

尽管先前的伪代码是以字符串输入和输出的形式编写的，但从概念上讲，用户提供的map和reduce函数有关联的类型：

```
map (k1,v1) → list(k2,v2)
reduce (k2,list(v2)) → list(v2)
```

即，输入键和值的域与输出键和值的域不同。此外，中间键和值的域与输出键和值的域相同。

我们的C++实现将字符串传递给用户定义的函数，并让用户代码在字符串和适当类型之间进行转换。

## 2.3 更多示例

以下是一些有趣的程序的简单示例，这些程序可以轻松地表示为MapReduce计算。

分布式Grep：map函数如果匹配到一个提供的模式就发出一行。reduce函数是一个恒等函数，它只是将提供的中间数据复制到输出。

URL访问频率计数：map函数处理网页请求日志，并输出hURL, 1i。reduce函数将同一URL的所有值相加，并发出一个hURL,总数i对。

反向网页链接图：map函数为在名为source的页面中找到的指向目标URL的每个链接输出htarget, sourcei对。reduce函数将所有与给定目标URL相关联的源URL列表连接起来，并发出一对：htarget, list(source)i。

每个主机的术语向量：术语向量将一个文档或一组文档中出现的最重要的词汇总结为hword, frequencyi对的列表。map函数为每个输入文档发出一对hhostname, term vectori（其中hostname从文档的URL中提取）。reduce函数被传递给一个给定主机的所有每个文档的术语向量。它将这些术语向量加在一起，丢弃不常见的术语，然后发出最终的hhostname, term vectori对。

倒排索引：map函数解析每个文档，并发出一系列hword, document IDi对。reduce函数接受一个给定单词的所有对，排序相应的文档ID并发出hword, list(document ID)i对。所有输出对的集合形成一个简单的倒排索引。很容易增加这个计算以跟踪单词位置。

分布式排序：map函数从每个记录中提取键，并发出hkey, recordi对。reduce函数发出所有不变的对。这个计算依赖于在第4.1节中描述的分区设施和在第4.2节中描述的排序属性。