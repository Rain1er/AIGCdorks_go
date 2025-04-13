# Record the learning process

## 2025-4-13
并发编程无非是解决 同步、互斥、进程通信、多线程执行 这4个问题
对应的关键字是
* sync.wait
* sync.mutex
* channel
* goruntime

查看了httpx的代码实现，有如下学习体会
1. 发现似乎还是一个goroutine负责一个请求，只要在声明waitGroup时指定最大并发量即可
```go
wg, _ := syncutil.New(syncutil.WithSize(r.options.Threads))
```
2. go多线程编程中若涉及到routine通信，则需要使用channel.尽量不要使用内存通信，因为内存通信会消耗大量内存，而channel则不会