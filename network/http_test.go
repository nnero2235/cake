package network

import (
	"cake/util"
	"strconv"
	"sync"
	"testing"
)

func TestHttpEngine_Get(t *testing.T) {
	httpEngine := CreateEngine()
	_, e := httpEngine.Get("https://www.baidu.com",nil)
	if e != nil {
		t.Errorf("%v",e)
	}
}

func TestHttpEngine_Get_Retry(t *testing.T) {
	httpEngine := CreateEngine()
	_,e := httpEngine.Get("https://www.oschina.net/project/tag/ff",nil)
	if e != nil {
		t.Errorf("%v",e)
	}
}

func TestHttpEngine_Download(t *testing.T) {
	httpEngine := CreateEngine()
	info := &DownloadInfo{
		Url:                "https://c-ssl.duitang.com/uploads/item/201412/25/20141225204152_aYEc3.jpeg",
		FilePath:           util.GetOSFilePath("tmp","go","cake_test","image"),
		FileName:           "1.jpg",
		HttpHeaders:        nil,
		DownloadWhenExists: true,
	}
	result := httpEngine.Download(info)
	if result.E != nil {
		t.Errorf("%v",result.E)
	} else {
		logger.InfoF("Success: %s -> %d ",result.FileFullName,result.FileSize)
	}
}

func TestHttpEngine_Concurrent_Download(t *testing.T) {
	httpEngine := CreateEngine()
	wg := sync.WaitGroup{}
	for i := 0; i<10 ; i++  {
		wg.Add(1)
		info := &DownloadInfo{
			Url:                "https://c-ssl.duitang.com/uploads/item/201412/25/20141225204152_aYEc3.jpeg",
			FilePath:           util.GetOSFilePath("tmp","go","cake_test","image"),
			FileName:           strconv.Itoa(i)+".jpg",
			HttpHeaders:        nil,
			DownloadWhenExists: true,
		}
		go func(info *DownloadInfo) {
			result := httpEngine.Download(info)
			if result.E != nil {
				t.Errorf("%v",result.E)
			} else {
				logger.InfoF("Success: %s -> %s ",result.FileFullName,util.GetFormatFileSize(result.FileSize))
			}
			wg.Done()
		}(info)
	}
	wg.Wait()
}

