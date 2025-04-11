package runner

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

func Exec() {
	for _, dork := range Dorks {
		token := getToken()
		if Target != "" {
			query(fmt.Sprintf("%s %s", Target, dork), token)
		} else {
			query(dork, token) // no target ,only use dork
		}
	}
}

func query(dork string, token string) {

	for i := 1; i <= 10; i++ {
		if i == responseData.TotalCount/100 {
			break
		}
		// 构造请求，GET参数固定写法
		if status == 403 {
			color.Red("[!] %s token is limited, change another token", token)
			token = getToken()
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
		proxyURL, _ := url.Parse("http://127.0.0.1:8080")

		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		Client := http.Client{
			Transport: transport,
			Timeout:   30 * time.Second, // 例如：设置 30 秒的总超时
		}
		resp, err := Client.Do(req)
		if err != nil {
			color.Red("GET faild")
			break
		}
		// read the resp
		status = resp.StatusCode
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			color.Red("response io error")
			break
		}
		defer resp.Body.Close() // 确保在函数结束时关闭 resp.Body

		// 解析json到结构体
		err = json.Unmarshal([]byte(body), &responseData)
		if err != nil {
			// handle potential errors during JSON parsing
			color.Red("json parse error!")
			os.Exit(0)
		}
		// Iterate through the items and print each html_url
		for i, item := range responseData.Items {
			fmt.Printf("%d: %s\n", i+1, item.HtmlUrl)
		}
	}

}
