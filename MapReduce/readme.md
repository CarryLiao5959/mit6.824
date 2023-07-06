1. 完成一次文件交接：将coor的一个文件交付给worker
2. worker拿到文件后进行map操作并将中间值保存到local disk
3. 当所有的map任务完成后，开始执行reduce任务
4. 规范代码
5. map任务全部完成后开始分配reduce任务，reduce任务结束后程序自动退出