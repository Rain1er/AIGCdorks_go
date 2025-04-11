package runner

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type Response struct {
	TotalCount int     `json:"total_count"`
	Items      []Items `json:"items"`
}
type Items struct {
	HtmlUrl string `json:"html_url"`
}

// 响应数据处理
var responseData Response
var status int
var currentPage int
var targetUrl []string // 目标url切片，等下去正则匹配里面的key

func Exec() {
	for _, dork := range Dorks {
		token := getToken()
		if Target != "" {
			query(fmt.Sprintf("%s %s", Target, dork), token)
		} else {
			query(dork, token) // no target ,only use dork
		}
	}

	// 分片处理 URL
	chunkSize := 10 // 每个 goroutine 处理的 URL 数量
	var wg sync.WaitGroup

	for i := 0; i < len(targetUrl); i += chunkSize {
		end := i + chunkSize
		if end > len(targetUrl) {
			end = len(targetUrl)
		}
		urlChunk := targetUrl[i:end]

		wg.Add(1)
		go func(urls []string) {
			defer wg.Done()
			processUrls(urls)
		}(urlChunk)
	}

	wg.Wait()
}

func query(dork string, token string) {

	for i := 1; i <= 10; i++ {
		currentPage = i
		if i == responseData.TotalCount/100 {
			break
		}
		// 构造请求，GET参数固定写法
		if status == 403 {
			color.Red("[!] %s token is limited, change another token", token)
			token = getToken()
			color.Red("----------------now token sequence is change to %d----------------", TokenSeq+1)
			i-- // bug 继续请求当前页
			continue
		}
		uri, _ := url.Parse("https://api.github.com/search/code")

		param := url.Values{}
		param.Set("q", dork)
		param.Set("per_page", strconv.Itoa(100)) //Integer to ASCII
		param.Set("page", strconv.Itoa(i))       // total_count / 100 ，max = 10
		uri.RawQuery = param.Encode()

		req, _ := http.NewRequest("GET", uri.String(), nil)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		// sent request
		proxyURL, _ := url.Parse("http://127.0.0.1:7890")

		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		Client := http.Client{
			Transport: transport,
			Timeout:   30 * time.Second, // 例如：设置 30 秒的总超时
		}
		resp, err := Client.Do(req)
		if err != nil {
			color.Red("request failed")
			continue
		}
		// read the resp
		status = resp.StatusCode
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			color.Red("response io error")
			continue
		}
		defer resp.Body.Close() // 确保在函数结束时关闭 resp.Body

		// 解析json到结构体
		err = json.Unmarshal([]byte(body), &responseData)
		if err != nil {
			// handle potential errors during JSON parsing
			color.Red("json parse error!")
			continue
		}

		// bug 由于风控，可能出现没有item的情况，需要重试
		if len(responseData.Items) == 0 {
			color.Red("[!] no items, retry")
			i--
			continue
		}
		// Iterate through the items and print each html_url
		for i, item := range responseData.Items {
			color.Yellow("共%d页，当前页面：%d\n", responseData.TotalCount/100, currentPage)
			fmt.Printf("%d: %s\n", i+1, item.HtmlUrl)
			targetUrl = append(targetUrl, item.HtmlUrl)
		}
	}
}

func processUrls(urls []string) {
	total := len(urls)
	for i, url := range urls {
		// 打印进度
		progress := float64(i+1) / float64(total) * 100
		fmt.Printf("Progress: %.2f%% (%d/%d) - Processing URL: %s\n", progress, i+1, total, url)

		// 使用 http.Get 获取网页内容
		resp, err := http.Get(url)
		if err != nil {
			color.Red("get url error: %v", err)
			continue
		}
		defer resp.Body.Close()

		// 读取网页内容
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			color.Red("read body error: %v", err)
			continue
		}

		// 检查 HTTP 状态码
		if resp.StatusCode == 200 {
			// 使用正则匹配 sk-or-v1-[a-z0-9]{64}
			re := regexp.MustCompile(`sk-or-v1-[a-z0-9]{64}`) // todo 匹配AGIC的顺序，因为api搜索似乎不支持正则
			matches := re.FindAllString(string(body), -1)     // 查找所有匹配项

			// 使用 map 去重
			uniqueMatches := make(map[string]bool)
			for _, match := range matches {
				uniqueMatches[match] = true
			}

			for match := range uniqueMatches {
				color.Green("[+] get key: %s", match)

				// 线程安全地写入文件
				if err := writeToFile("key.txt", match); err != nil {
					color.Red("write to file error: %v", err)
				}
			}
		}
	}
}

// writeToFile 线程安全地将内容写入文件
func writeToFile(filename, content string) error {
	// 使用互斥锁确保文件写入的线程安全
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	// 打开文件（追加模式）
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file error: %v", err)
	}
	defer file.Close()

	// 写入内容并添加换行符
	if _, err := file.WriteString(content + "\n"); err != nil {
		return fmt.Errorf("write file error: %v", err)
	}

	return nil
}
