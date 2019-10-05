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
		url:                "https://c-ssl.duitang.com/uploads/item/201412/25/20141225204152_aYEc3.jpeg",
		filePath:           util.GetOSFilePath("tmp","go","cake_test","image"),
		fileName:           "1.jpg",
		httpHeaders:        nil,
		downloadWhenExists: true,
	}
	result := httpEngine.Download(info)
	if result.e != nil {
		t.Errorf("%v",result.e)
	} else {
		logger.InfoF("Success: %s -> %d ",result.fileFullName,result.fileSize)
	}
}

func TestHttpEngine_Concurrent_Download(t *testing.T) {
	httpEngine := CreateEngine()
	wg := sync.WaitGroup{}
	for i := 0; i<10 ; i++  {
		wg.Add(1)
		info := &DownloadInfo{
			url:                "https://c-ssl.duitang.com/uploads/item/201412/25/20141225204152_aYEc3.jpeg",
			filePath:           util.GetOSFilePath("tmp","go","cake_test","image"),
			fileName:           strconv.Itoa(i)+".jpg",
			httpHeaders:        nil,
			downloadWhenExists: true,
		}
		go func(info *DownloadInfo) {
			result := httpEngine.Download(info)
			if result.e != nil {
				t.Errorf("%v",result.e)
			} else {
				logger.InfoF("Success: %s -> %s ",result.fileFullName,util.GetFormatFileSize(result.fileSize))
			}
			wg.Done()
		}(info)
	}
	wg.Wait()
}

