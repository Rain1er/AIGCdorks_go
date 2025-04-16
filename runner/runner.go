package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	syncutil "github.com/projectdiscovery/utils/sync"
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

// Response data processing
var responseData Response
var status int
var currentPage int
var targetUrl []string // 目标url切片，等下去正则匹配里面的key

// 全局变量和互斥锁
var processedCount int
var processedCountMutex sync.Mutex

// 新增：定义一个通道用于存储匹配结果
var matchChan = make(chan string, 100) // 缓冲通道，容量为100

// 类似于构造函数
func init() {
	// 新增：启动一个 goroutine 用于从通道中读取数据并写入文件
	go func() {
		for match := range matchChan {
			if err := writeToFile("./source/key", match); err != nil {
				color.Red("write to file error: %v", err)
			}
		}
	}()
}

func Exec() {
	for _, dork := range Dorks {
		token := updateToken() // 得到第一个token
		if Target != "" {
			// 指定目标域的搜索
			reqAndParse(fmt.Sprintf("%s %s", Target, dork), token)
		} else {
			reqAndParse(dork, token) // no target ,only use dork
		}
	}

	// sharding ProcessingURL,设置最大并发
	wg, _ := syncutil.New(syncutil.WithSize(Threads))

	for _, u := range targetUrl {
		wg.Add()
		go func(url string) {
			defer wg.Done()
			processUrls(url)
		}(u)
	}

	wg.Wait()

	// Read key.txt to deduplicate each line
	_ = removeDuplicatesFromFile("./source/key")

}

func reqAndParse(dork string, token string) {
	firstIn := true
	for i := 1; i <= 10; i++ {
		currentPage = i
		// 不是第一次循环，那么需要判断页数
		if !firstIn {
			// 如果只有128项呢？应该获取第1 2页才对
			if responseData.TotalCount-100*(i-1) <= 0 {
				break
			}
		}
		firstIn = false

		// 上来就要判断上次的请求是否limit，如果是就换token
		if status == 403 {
			color.Red("[error] %s token is limited, change another token", token)
			token = updateToken()

			color.Red("----------------now token sequence is change to %d----------------", TokenSeq)
			i--        // 还是请求上次页号
			status = 1 // 要更新status 否则会死循环
			continue
		}

		// 构造请求，GET参数固定写法
		uri, _ := url.Parse("https://api.github.com/search/code")

		param := url.Values{}
		param.Set("q", dork)                         // todo 增加各种语言分类，减少检索结果，从而绕过1000条结果限制
		param.Set("per_page", strconv.Itoa(100))     // Integer to ASCII
		param.Set("page", strconv.Itoa(currentPage)) // total_count / 100 ，max = 10
		uri.RawQuery = param.Encode()

		req, _ := http.NewRequest("GET", uri.String(), nil)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		// sent request
		proxyURL, _ := url.Parse("http://127.0.0.1:8080")

		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		Client := http.Client{
			Transport: transport,
			Timeout:   10 * time.Second, // 设置 10 秒的总超时
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
			color.Red("[error] no items, retry")
			i--
			continue
		}

		// Iterate through the items and print each html_url
		for i, item := range responseData.Items {
			color.Yellow("共%d页，当前页面：%d\n", responseData.TotalCount/100+1, currentPage)
			fmt.Printf("%d: %s\n", i+1, item.HtmlUrl)
			targetUrl = append(targetUrl, item.HtmlUrl)
		}

		uniqueUrl := make(map[string]bool)
		for _, u := range targetUrl {
			uniqueUrl[u] = true
		}
		targetUrl = []string{}

		for u := range uniqueUrl {
			targetUrl = append(targetUrl, u)
		}
	}

}

func processUrls(url string) {
	// 打印进度
	processedCountMutex.Lock()
	processedCount++
	currentProcessedCount := processedCount
	processedCountMutex.Unlock()

	progress := float64(currentProcessedCount) / float64(len(targetUrl)) * 100
	fmt.Printf("Progress: %.2f%% (%d/%d) - Processing URL: %s\n", progress, currentProcessedCount, len(targetUrl), url)

	// 使用 http.Get 获取网页内容
	resp, err := http.Get(url)
	if err != nil {
		color.Red("get url error: %v", err)
		return
	}
	defer resp.Body.Close()

	// 读取网页内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		color.Red("read body error: %v", err)
		return
	}

	// 检查 HTTP 状态码
	if resp.StatusCode == 200 {
		// 使用正则匹配 sk-or-v1-[a-z0-9]{64}
		re := regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`) // todo 匹配AGIC的顺序，因为api搜索似乎不支持正则
		matches := re.FindAllString(string(body), -1)  // 查找所有匹配项

		// 使用 map 去重
		uniqueMatches := make(map[string]bool)
		for _, match := range matches {
			uniqueMatches[match] = true
		}

		for match := range uniqueMatches {
			color.Green("[+] get key: %s", match)

			// 修改：将匹配结果写入通道
			matchChan <- match
		}
	}
}

// writeToFile 线程安全地将内容写入文件，注意互斥锁要在外面定义
func writeToFile(filename, content string) error {
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

// removeDuplicatesFromFile 读取文件内容，去除重复行，然后将结果写回文件
func removeDuplicatesFromFile(filename string) error {
	// 使用 map 来存储唯一的行
	uniqueLines := make(map[string]bool)

	// 打开文件进行读取
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file error: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() { // 扫描每一行
		line := scanner.Text()
		uniqueLines[line] = true
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read file error: %v", err)
	}

	// 打开文件进行写入（覆盖模式）
	file, err = os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file error: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for line := range uniqueLines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("write file error: %v", err)
		}
	}

	// 确保所有内容都被写入文件
	return writer.Flush()
}
