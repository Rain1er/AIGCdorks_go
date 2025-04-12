package check

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

func CheckOpenRouter(f *os.File) {
	// 从key中读入每一行放到切片中
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// 1. 验证key是否有效
	var wg sync.WaitGroup
	for i := 0; i < len(lines); i++ {
		wg.Add(1)
		go func(key string) {
			defer wg.Done() // 确保 wg.Done() 在 goroutine 结束时调用

			// 构造请求，GET参数固定写法
			uri, _ := url.Parse("https://openrouter.ai/api/v1/auth/key")
			req, _ := http.NewRequest("GET", uri.String(), nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key)) // 每个 goroutine 使用独立的 req

			// 设置代理和超时
			proxyURL, _ := url.Parse("http://127.0.0.1:8080")
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			client := http.Client{
				Transport: transport,
				Timeout:   10 * time.Second, // 设置 10 秒的总超时
			}

			// 发送请求
			resp, err := client.Do(req)
			if err != nil {
				color.Red("verify api failed: %v", err)
				return // 直接结束线程
			}
			defer resp.Body.Close() // 安全关闭响应体

			// 返回状态码为200表示key有效
			if resp.StatusCode == http.StatusOK {
				color.Green(key)
			}
		}(lines[i])
		// 这里需要等待设置等待，否则会因为并发量过高导致有些请求失败，引入计数，每10个请求暂停一下
		if i%10 == 0 {
			time.Sleep(time.Second * 5)
		}
		// 或许这里可以参考前面的实现，在一个goruntime中完成多个http请求，避免创建过过多goruntime
	}

	wg.Wait()

}
