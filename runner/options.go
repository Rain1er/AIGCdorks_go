package runner

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"os"
	"strings"
)

// 变量名以大写开头，可以在包外访问，类似于public

// 菜单参数
var Target string
var DorkFile string
var TokenFile string
var Delay int64

// 所有token和dork
var Tokens []string
var Dorks []string
var TokenSeq = 0

func Menu() {
	flag.StringVar(&TokenFile, "tf", "", "github personal access token file")
	flag.StringVar(&DorkFile, "df", "", "github dorks file path")
	flag.StringVar(&Target, "t", "", "target which search in github")
	flag.Int64Var(&Delay, "d", 10, "how many seconds does it wait each time")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// 判断是否输入参数
	if flag.NFlag() == 0 { // 使用的命令行参数个数，这个地方可以用来判断用户是否输入参数（程序正常情况下会使用默认参数）
		flag.Usage()
		os.Exit(0)
	}
	// 判断是否有关键词
	if DorkFile == "" {
		color.Red("require dorkfile")
		os.Exit(0)
	}
}

/*
从文件中读取token和dork并放到切片中
*/
func Parseparam() {
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
	f1, err := os.ReadFile(DorkFile)
	if err != nil {
		color.Red("file error: %v", err)
		os.Exit(0)
	}
	// 最后一行可能有空字符
	dorks := strings.Split(string(f1), "\n")
	if dorks[len(dorks)-1] == "" {
		dorks = dorks[:len(dorks)-1]
	}
	Dorks = dorks

	color.Blue("[+] got %d tokens and %d dorks\n\n", len(tokens), len(dorks))
}

/*
多个token轮询，直到一个token达到限制之后，再切换下一个token。如此循环即可
*/
func updateToken() string {
	token := Tokens[TokenSeq]
	TokenSeq += 1
	if len(Tokens) == TokenSeq {
		TokenSeq = 0
	}
	return token
}
