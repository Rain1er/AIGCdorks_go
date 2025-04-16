package check

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	syncutil "github.com/projectdiscovery/utils/sync"
	"net/http"
	"net/url"
	"os"
	"time"
)

func Check302AI(f *os.File) {
	// 从key中读入每一行放到切片中
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// 1. 验证key是否有效
	wg, _ := syncutil.New(syncutil.WithSize(50))

	// todo 可以优化成channel
	for i := 0; i < len(lines); i++ {
		wg.Add()
		go func(key string) {
			defer wg.Done() // 确保 wg.Done() 在 goroutine 结束时调用

			// 构造请求，GET参数固定写法
			uri, _ := url.Parse("https://api.302.ai/v1/chat/completions")
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
			if resp.StatusCode == 503 {
				color.Green(key)

			}
		}(lines[i])
	}

	wg.Wait()

}
