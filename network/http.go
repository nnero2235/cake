package network

import (
	"cake/util"
	"cake/util/log"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

var logger = log.GetLogger()

const (
	Timeout        = 30 * 1000 * 1000 * 1000 //nano seconds
	MaxConnections = 50
	Retries        = 3
)

//HttpEngine should be a single instance for the specified business
//it means every business should have different engine
type HttpEngine struct{
	maxConnections int //concurrent number
	timeout time.Duration //timeout for all time cost until response was fully read
	retries int //when fail in any issue,retry times
	client *http.Client //do real http network
	sem chan struct{} //for control concurrent go route
}

func CreateEngine() *HttpEngine{
	return CreateEngineByParams(MaxConnections, Timeout, Retries)
}

func CreateEngineByParams(maxConnections int, timeout time.Duration, retries int) *HttpEngine{
	client := &http.Client{
		Timeout: timeout,
	}
	engine := &HttpEngine{
		maxConnections: maxConnections,
		timeout:        timeout,
		retries:        retries,
		client:         client,
		sem:            make(chan struct{},maxConnections),
	}
	return engine
}

//Get method to fetch all content into string
//sync network request in common go route
func (engine *HttpEngine) Get(url string,headers map[string]string) (string,error){
	engine.sem <- struct{}{} //get sem if full , that will block
	defer func(){ <- engine.sem}() // function end, sem is returned

	request, e := http.NewRequest("GET", url, nil)
	if e != nil {
		return "",e
	}
	//need add header
	if headers != nil && len(headers) > 0 {
		for k,v := range headers {
			logger.DebugF("[GET] url: %s Add Header: [%s : %s]",url,k,v)
			request.Header.Set(k,v)
		}
	}
	retryCount := 0
RETRY_LOOP:
	response, e := engine.client.Do(request)
	if e != nil {
		logger.WarnF("[GET] Retries: %d -> Http url: \"%s\" Error: %v",retryCount,url,e)
		retryCount += 1
		if retryCount >= engine.retries{
			return "",fmt.Errorf("[GET] Retry \""+strconv.Itoa(engine.retries)+
				"\" but still can't Get url: "+url+" status: "+strconv.Itoa(response.StatusCode))
		}
		time.Sleep(util.GetTimeSecond(1))
		goto RETRY_LOOP
	}
	if response.StatusCode == 200 {
		bytes, e := ioutil.ReadAll(response.Body)
		if e != nil {
			logger.WarnF("[GET] Retries: %d -> Http ReadAll: \"%s\" Error: %v",retryCount,url,e)
			retryCount += 1
			if retryCount >= engine.retries{
				return "",fmt.Errorf("[GET] Retry \""+strconv.Itoa(engine.retries)+
					"\" but still can't Read url: "+url+" All Data, status: "+strconv.Itoa(response.StatusCode))
			}
			e := response.Body.Close()
			if e != nil {
				logger.WarnF("[GET] Http Get Response Close Error: %v",e)
			}
			time.Sleep(util.GetTimeSecond(1))
			goto RETRY_LOOP
		}
		logger.Info("[GET] 200 -> "+url)
		defer func() {
			e := response.Body.Close()
			if e != nil {
				logger.WarnF("[GET] Http Get Response Close Error: %v",e)
			}
		}()
		return string(bytes),nil
	} else {
		logger.WarnF("[GET] Retries: %d -> Http Status: %d",retryCount,response.StatusCode)
		retryCount += 1
		if retryCount >= engine.retries{
			return "",fmt.Errorf("[GET] Retry \""+strconv.Itoa(engine.retries)+
				"\" but still can't Get url: "+url+" status: "+strconv.Itoa(response.StatusCode))
		}
		e := response.Body.Close()
		if e != nil {
			logger.WarnF("[GET] Http Get Response Close Error: %v",e)
		}
		time.Sleep(util.GetTimeSecond(1))
		goto RETRY_LOOP
	}
}

const defaultDownloadBufferSize int = 8196

//for download prepare info
type DownloadInfo struct {
	url string
	filePath string
	fileName string
	httpHeaders map[string]string
	downloadWhenExists bool //if file exists,delete old one and download new one
}

//for download return data
type DownloadResult struct {
	url string
	fileFullName string
	fileSize int64 //n bytes
	e error //nil means success,other means problem happened
}

func downloadInfoValid(info *DownloadInfo) error {
	if info == nil {
		return fmt.Errorf("info is nil. Nothing to Download")
	}
	if info.url == "" {
		return fmt.Errorf("url is nil. Nothing to Download")
	}
	if info.fileName == "" {
		return fmt.Errorf("fileName is nil. Nothing to Download")
	}
	if info.filePath == "" {
		return fmt.Errorf("filePath is nil. Nothing to Download")
	}
	return nil
}

func filePathValid(info *DownloadInfo) error {
	if _, e := os.Open(info.filePath); os.IsNotExist(e){
		if e := os.MkdirAll(info.filePath, util.FileRWOwner);e != nil{
			return fmt.Errorf("[Download] Create Dir Fail: %v ", e)
		}
		logger.InfoF("[Download] file Path: \"%s\" doesn't exists.Create it!",info.filePath)
	}
	return nil
}

func createFileWriter(info *DownloadInfo,fileFullName string) (*os.File,error) {
	if info.downloadWhenExists {
		file, e := os.OpenFile(fileFullName,os.O_CREATE|os.O_WRONLY|os.O_TRUNC,util.FileRWRAll)
		if e != nil{
			return nil,fmt.Errorf("Create File: %s failed! Error: %v ",fileFullName,e)
		}
		return file,nil
	} else {
		file, e := os.OpenFile(fileFullName,os.O_RDONLY,util.FileRWRAll)
		if os.IsExist(e){
			e = file.Close()
			if e != nil{
				logger.Warn("File: "+fileFullName+" close error: "+e.Error())
			}
			return nil,fmt.Errorf("File: %s already exists. skip download",fileFullName)
		}
		file, e = os.OpenFile(fileFullName,os.O_CREATE|os.O_WRONLY,util.FileRWRAll)
		if e != nil{
			return nil,fmt.Errorf("Create File: "+fileFullName+" failed! Error: %v",e)
		}
		return file,nil
	}
}

//download to local file.Use Get method.
//consider Get and Download should or not be a same instance when use.
//return none nil result
func (engine *HttpEngine) Download(info *DownloadInfo) *DownloadResult {
	result := &DownloadResult{}
	e := downloadInfoValid(info)
	if e != nil {
		result.e = e
		return result
	}
	e = filePathValid(info)
	if e != nil {
		result.e = e
		return result
	}

	result.url = info.url
	result.fileFullName = info.filePath+string(os.PathSeparator)+info.fileName

	//open file to write
	file,e := createFileWriter(info,result.fileFullName)
	if e != nil {
		result.e = e
		return result
	}
	//close defer
	defer func(fullName string) {
		e := file.Close()
		if e != nil {
			logger.Warn("File: "+fullName+" close error: "+e.Error())
		}
	}(result.fileFullName)

	engine.sem <- struct{}{} //get sem if full , that will block
	defer func(){ <- engine.sem}() // function end, sem is returned

	request, e := http.NewRequest("GET", info.url, nil)
	if e != nil {
		result.e = e
		return result
	}
	//need add header
	if info.httpHeaders != nil && len(info.httpHeaders) > 0 {
		for k,v := range info.httpHeaders {
			logger.DebugF("[Download] url: %s Add Header: [%s : %s]",info.url,k,v)
			request.Header.Set(k,v)
		}
	}

	retryCount := 0
RETRY_LOOP:
	response, e := engine.client.Do(request)
	if e != nil {
		logger.WarnF("[Download] Retries: %d -> url: \"%s\" Error: %v",retryCount,info.url,e)
		retryCount += 1
		if retryCount >= engine.retries{
			e = fmt.Errorf("[Download] Retry \""+strconv.Itoa(engine.retries)+
				"\" but still can't Download url: "+info.url+" status: "+strconv.Itoa(response.StatusCode))
			result.e = e
			return result
		}
		time.Sleep(util.GetTimeSecond(2))
		goto RETRY_LOOP
	}
	if response.StatusCode == 200 {
		buffer := make([]byte, defaultDownloadBufferSize)
		for {
			n, e := response.Body.Read(buffer)
			if e != nil {
				if e == io.EOF { //read over
					break
				}
				logger.WarnF("[Download] Retries: %d -> url: \"%s\" Read %d bytes Error: %v", retryCount, info.url, result.fileSize, e)
				retryCount += 1
				if retryCount >= engine.retries {
					e = fmt.Errorf("[Download] Retry \"" + strconv.Itoa(engine.retries) +
						"\" but still can't Read url: " + info.url + " -> already read %d bytes Data, status: " + strconv.Itoa(response.StatusCode))
					result.e = e
					return result
				}
				e := response.Body.Close()
				if e != nil {
					logger.WarnF("[Download] Http Get Response Close Error: %v", e)
				}
				time.Sleep(util.GetTimeSecond(2))
				goto RETRY_LOOP
			}
			if n == 0 { //nothing to read
				break
			}
			wn, e := file.Write(buffer[:n])
			if e != nil { //write error.just return. no retry
				result.e = e
				return result
			}
			logger.TraceF("[Download] url:%s -> read %d bytes. Write %d bytes", info.url, n,wn)
			result.fileSize += int64(wn)
		}
		logger.InfoF("[Download] 200 -> %s fileSize: %s", info.url, util.GetFormatFileSize(result.fileSize))
		defer func() {
			e := response.Body.Close()
			if e != nil {
				logger.WarnF("[Download] Http Get Response Close Error: %v", e)
			}
		}()
		return result
	} else {
		logger.WarnF("[Download] Retries: %d -> Download Status: %d", retryCount, response.StatusCode)
		retryCount += 1
		if retryCount >= engine.retries {
			e = fmt.Errorf("[Download] Retry \"" + strconv.Itoa(engine.retries) +
				"\" but still can't Download url: " + info.url + " status: " + strconv.Itoa(response.StatusCode))
			result.e = e
			return result
		}
		e := response.Body.Close()
		if e != nil {
			logger.WarnF("[Download] Http Get Response Close Error: %v", e)
		}
		time.Sleep(util.GetTimeSecond(1))
		goto RETRY_LOOP
	}
}