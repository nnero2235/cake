package util

import (
	"testing"
	"time"
)

func TestGetTimeSecond(t *testing.T) {
	defer FuncElapsed("Get Time Second")()
	time.Sleep(GetTimeSecond(5))
}
