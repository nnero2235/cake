package network

import (
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
		filePath:           "D:\\tmp\\go\\cake_test\\image",
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

