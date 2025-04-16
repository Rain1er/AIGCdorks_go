package check

import (
	"os"
	"testing"
)

func TestCheck302AI(t *testing.T) {
	f, err := os.Open("../source/key")
	if err != nil {
		t.Errorf("OpenFile() error = %v", err)
		return
	}
	defer f.Close()
	Check302AI(f)
}
