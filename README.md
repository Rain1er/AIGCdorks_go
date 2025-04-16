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
httpx中的相关代码
```go
wg, _ := syncutil.New(syncutil.WithSize(r.options.Threads))
```
2. go多线程编程中若涉及到routine通信，则需要使用channel.尽量不要使用内存通信，因为内存通信会消耗大量内存，而channel则不会



## 2025-4-16
bugs
修正分页的逻辑
1. 使用flag标记是否第一次进入循环，感觉比do while好用
2. 在循环开始就判断已检索的数量

todo
1. 完成多个apikey的收集(整合各类api的调用，流式输出)
2. 分别验证apikey的有效性
3. 结合oneapi实现调用的负载均衡
4. 寻找更多api供应商
5. 为了方便起见，直接使用PD的库设置最大并发吧？顺便看看他是怎么做的，直接拿函数过来似乎更加轻便
6. 数据持久化
