package util

import (
	"cake/util/log"
	"runtime"
	"testing"
)

var logger = log.GetLogger()

func TestGetOSFilePath(t *testing.T) {
	path := GetOSFilePath("tmp", "go", "cake")
	logger.Info(path)
	if runtime.GOOS == "windows" && path != "D:\\tmp\\go\\cake" {
		t.Error("wrong path:"+path+" at os:"+runtime.GOOS)
	} else if runtime.GOOS != "windows" && path != "/tmp/go/cake" {
		t.Error("wrong path:"+path+" at os:"+runtime.GOOS)
	}
}

func TestGetFileSize(t *testing.T) {
	var nBytes int64 = 936
	logger.Info(GetFormatFileSize(nBytes))
	kb := 22440
	logger.Info(GetFormatFileSize(int64(kb)))
	mb := 36722600
	logger.Info(GetFormatFileSize(int64(mb)))
	var gb int64 = 58018781096
	logger.Info(GetFormatFileSize(gb))
}