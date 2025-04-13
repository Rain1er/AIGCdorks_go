package check

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// parse json, if it is a trial user, don't need this key
// 封装 和 解耦。  T 结构体提供了一个顶层的容器，将 data 结构体组织起来。
// 处理 API 响应的整体结构: API 响应通常不仅仅包含数据 (data)。 它们可能还包含状态码、错误信息、元数据等。
type T struct {
	Data struct {
		Label             string      `json:"label"`
		Limit             interface{} `json:"limit"`
		Usage             float64     `json:"usage"`
		IsProvisioningKey bool        `json:"is_provisioning_key"`
		LimitRemaining    interface{} `json:"limit_remaining"`
		IsFreeTier        bool        `json:"is_free_tier"`
		RateLimit         struct {
			Requests int    `json:"requests"`
			Interval string `json:"interval"`
		} `json:"rate_limit"`
	} `json:"data"`
}

func CheckOpenRouter(f *os.File) {
	// 从key中读入每一行放到切片中
	// todo 并发场景使用channel可能会更好？不需要单独对切片进行分片操作了
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
				var result T
				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					color.Red("failed to parse json: %v", err)
					return
				}

				// 判断 is_free_tier 字段是否为 false
				if !result.Data.IsFreeTier {
					color.Green(key)
				}
			}
		}(lines[i])

		// 这里需要等待设置等待，否则会因为并发量过高导致有些请求失败，引入计数，每10个请求暂停一下
		if i%10 == 0 {
			time.Sleep(time.Second * 5)
		}

		// 或许这里可以参考前面的实现，在一个goruntime中完成多个http请求，避免创建过过多goruntime

		// 还需要增加openai/gpt-4-turbo-preview的一次调用，判断这个key是否有余额

		// 研究发现or充值的人并不多啊，要改成别的试试了

	}

	wg.Wait()

}
