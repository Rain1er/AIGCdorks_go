package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
)

// 菜单参数
// 变量名以大写开头，可以在包外访问，类似于public
var Target string
var DorkFile string
var Keyword string
var TokenFile string
var NeedWaitSecond int64
var Delay int64

// 所有token和dork
var TokenSeq = 0
var Tokens []string
var Dorks []string

// 错误次数，超过100次就强行结束程序，避免一直运行卡死
var ErrorTimes = 0
var ErrorMaxTimes = 100

func query(dork string, token string) {
	// 构造请求，GET参数固定写法
	guri := "https://api.github.com/search/code"
	uri, _ := url.Parse(guri)

	param := url.Values{}
	param.Set("q", dork)
	param.Set("per_page", strconv.Itoa(100)) //Integer to ASCII
	param.Set("page", strconv.Itoa(10))
	uri.RawQuery = param.Encode()

	req, _ := http.NewRequest("GET", uri.String(), nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	Client := http.Client{}
	resp, err := Client.Do(req)

	// 结果判断
	if err != nil {
		color.Red("error: %v", err)
	} else {
		source, _ := os.ReadAll(resp.Body)
		var tmpSource map[string]jsoniter.Any
		_ = jsoniter.Unmarshal(source, &tmpSource)

		if tmpSource["documentation_url"] != nil { // 错误拦截
			// 错误次数太多，就直接停止程序了，避免一直卡死
			ErrorTimes += 1
			if ErrorTimes >= ErrorMaxTimes {
				color.Red("Too many errors, auto stop")
				os.Exit(0)
			}
			if NeedWait {
				color.Blue("error: %s ; and we need wait %ds", jsoniter.Get(source, "documentation_url").ToString(), NeedWaitSecond)
				time.Sleep(time.Second * time.Duration(NeedWaitSecond))
				token = getToken()
				query(dork, token)
			} else {
				color.Red("error: %s", jsoniter.Get(source, "documentation_url").ToString())
			}
		} else if tmpSource["total_count"] != nil { // 总数
			totalCount := jsoniter.Get(source, "total_count").ToInt()
			totalCountString := color.YellowString(fmt.Sprintf("(%s)", strconv.Itoa(totalCount)))
			uriString := color.GreenString(strings.Replace(uri.String(), "https://api.github.com/search/code", "https://github.com/search", -1) + "&s=indexed&type=Code&o=desc")
			fmt.Println(dork, " | ", totalCountString, " | ", uriString)
		} else { // 其他未知错误
			color.Blue("unknown error happened: %s", string(source))
		}
	}

}

func menu() {
	flag.StringVar(&DorkFile, "df", "", "github dorks file path")
	flag.StringVar(&TokenFile, "tf", "", "github personal access token file")
	flag.StringVar(&Keyword, "k", "", "github search keyword")
	flag.StringVar(&Target, "t", "", "target which search in github")
	flag.Int64Var(&Delay, "d", 10, "how many seconds does it wait each time")

	flag.Usage = func() {
		color.Green(`
 ____    __  __  __  __  ____  ___ 
(  _ \  /. |(  \/  )/  )(_  _)/ __)
 )(_) )(_  _))    (  )(   )(  \__ \
(____/   (_)(_/\/\_)(__) (__) (___/

                       v 0.1
`)
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// 判断是否输入参数
	if flag.NFlag() == 0 { // 使用的命令行参数个数，这个地方可以用来判断用户是否输入参数（程序正常情况下会使用默认参数）
		flag.Usage()
		os.Exit(0)
	}
	// 判断是否有目标
	if Target == "" {
		color.Red("require target")
		os.Exit(0)
	}
	// 判断是否有关键词
	if DorkFile == "" && Keyword == "" {
		color.Red("require keyword or dorkfile")
		os.Exit(0)
	}
}

/*
从文件中读取token和dork并放到切片中
*/
func parseparam() {
	// 解析token
	f, err := os.ReadFile(TokenFile)
	if err != nil {
		color.Red("TokenFile error: %v", err)
		os.Exit(0)
	}

	tokens := strings.Split(string(f), "\n")
	// 如果最后最后一项包含换行，那么会多出一个空项
	if tokens[len(tokens)-1] == "" {
		tokens = tokens[:len(tokens)-1]
	}
	Tokens = tokens // 提升作用域

	// 解析dork
	dkres, err := os.ReadFile(DorkFile)
	if err != nil {
		color.Red("file error: %v", err)
		os.Exit(0)
	}
	dorks := strings.Split(string(dkres), "\n")
	if dorks[len(dorks)-1] == "" {
		dorks = dorks[:len(dorks)-1]
	}
	Dorks = dorks

	color.Blue("[+] got %d tokens and %d dorks\n\n", len(tokens), len(dorks))
}

/*
多个token轮询，直到一个token达到限制之后，再切换下一个token。如此循环即可
*/
func getToken() string {
	token := Tokens[TokenSeq]
	TokenSeq += 1
	if len(Tokens) == TokenSeq {
		TokenSeq = 0
	}
	return token
}

func main() {
	menu()
	parseparam()

	for _, dork := range Dorks {
		token := getToken()
		query(fmt.Sprintf("%s %s", Target, dork), token)
	}
}
