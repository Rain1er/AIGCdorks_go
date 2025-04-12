package check

import (
	"os"
	"testing"
)

// TestCheckOpenRouter 测试 CheckOpenRouter 函数
func TestCheckOpenRouter(t *testing.T) {
	f, err := os.Open("../key")
	if err != nil {
		t.Errorf("OpenFile() error = %v", err)
		return
	}
	defer f.Close()

	CheckOpenRouter(f)
}
