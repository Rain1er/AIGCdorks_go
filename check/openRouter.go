package check

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	syncutil "github.com/projectdiscovery/utils/sync"
	"net/http"
	"net/url"
	"os"
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
	// 简单读取场景不需要使用channel，这里不涉及到进程通信
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// 1. 验证key是否有效
	wg, _ := syncutil.New(syncutil.WithSize(50)) // pd封装好的库，可以直接指定并发量

	for i := 0; i < len(lines); i++ {
		wg.Add()
		go func(key string) {
			defer wg.Done() // 确保 wg.Done() 在 goroutine 结束时调用

			// 构造请求，GET参数固定写法
			uri, _ := url.Parse("https://openrouter.ai/api/v1/auth/key")
			req, _ := http.NewRequest("GET", uri.String(), nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key)) // 每个 goroutine 使用独立的 req

			// 设置代理和超时，可查看http请求，用于调试等
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
		// 直接使用PD的库设置最大并发量了
		//if i%10 == 0 {
		//	time.Sleep(time.Second * 5)
		//}

	}

	wg.Wait()

}
