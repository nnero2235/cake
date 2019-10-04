package util

import (
	"cake/util/log"
	"fmt"
	"time"
)

//statistic func elapsed time
func FuncElapsed(name string) func(){
	now := time.Now()
	return func() {
		logger := log.GetLogger()
		timeStr := fmt.Sprintf("%s elapsed: %v", name, time.Since(now))
		logger.Trace(timeStr)
	}
}

//get time second simple method
func GetTimeSecond(s int) time.Duration{
	return time.Duration(s) * time.Second
}
