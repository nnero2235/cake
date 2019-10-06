package util

import (
	"cake/util/log"
	"fmt"
	"os"
)

var logger = log.GetLogger()

func GetParamsFromCMDBySpace() ([]string,error) {
	params := os.Args[1:]
	if len(params) == 0 {
		return nil,fmt.Errorf("params is nil.")
	} else {
		for _, k := range params {
			logger.InfoF("Get Param:[%s]", k)
		}
	}
	return params,nil
}
