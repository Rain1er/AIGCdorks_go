package main

import (
	"github.com/Rain1er/AIGCdorks_go/runner"
)

func main() {
	runner.Menu()
	runner.ParseParam()
	runner.Exec() // 获得apikey
}
