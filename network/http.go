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
			logger.DebugF("[GET] Url: %s Add Header: [%s : %s]",url,k,v)
			request.Header.Set(k,v)
		}
	}
	retryCount := 0
RETRY_LOOP:
	response, e := engine.client.Do(request)
	if e != nil {
		logger.WarnF("[GET] Retries: %d -> Http Url: \"%s\" Error: %v",retryCount,url,e)
		retryCount += 1
		if retryCount >= engine.retries{
			return "",fmt.Errorf("[GET] Retry \""+strconv.Itoa(engine.retries)+
				"\" but still can't Get Url: "+url+" status: "+strconv.Itoa(response.StatusCode))
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
					"\" but still can't Read Url: "+url+" All Data, status: "+strconv.Itoa(response.StatusCode))
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
				"\" but still can't Get Url: "+url+" status: "+strconv.Itoa(response.StatusCode))
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
	Url                string
	FilePath           string
	FileName           string
	HttpHeaders        map[string]string
	DownloadWhenExists bool //if file exists,delete old one and download new one
}

//for download return data
type DownloadResult struct {
	Url          string
	FileFullName string
	FileSize     int64 //n bytes
	E            error //nil means success,other means problem happened
}

func downloadInfoValid(info *DownloadInfo) error {
	if info == nil {
		return fmt.Errorf("info is nil. Nothing to Download")
	}
	if info.Url == "" {
		return fmt.Errorf("Url is nil. Nothing to Download")
	}
	if info.FileName == "" {
		return fmt.Errorf("FileName is nil. Nothing to Download")
	}
	if info.FilePath == "" {
		return fmt.Errorf("FilePath is nil. Nothing to Download")
	}
	return nil
}

func filePathValid(info *DownloadInfo) error {
	if _, e := os.Open(info.FilePath); os.IsNotExist(e){
		if e := os.MkdirAll(info.FilePath, util.FileRWOwner);e != nil{
			return fmt.Errorf("[Download] Create Dir Fail: %v ", e)
		}
		logger.InfoF("[Download] file Path: \"%s\" doesn't exists.Create it!",info.FilePath)
	}
	return nil
}

func createFileWriter(info *DownloadInfo,fileFullName string) (*os.File,error) {
	if info.DownloadWhenExists {
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
		result.E = e
		return result
	}
	e = filePathValid(info)
	if e != nil {
		result.E = e
		return result
	}

	result.Url = info.Url
	result.FileFullName = info.FilePath +string(os.PathSeparator)+info.FileName

	//open file to write
	file,e := createFileWriter(info,result.FileFullName)
	if e != nil {
		result.E = e
		return result
	}
	//close defer
	defer func(fullName string) {
		e := file.Close()
		if e != nil {
			logger.Warn("File: "+fullName+" close error: "+e.Error())
		}
	}(result.FileFullName)

	engine.sem <- struct{}{} //get sem if full , that will block
	defer func(){ <- engine.sem}() // function end, sem is returned

	request, e := http.NewRequest("GET", info.Url, nil)
	if e != nil {
		result.E = e
		return result
	}
	//need add header
	if info.HttpHeaders != nil && len(info.HttpHeaders) > 0 {
		for k,v := range info.HttpHeaders {
			logger.DebugF("[Download] Url: %s Add Header: [%s : %s]",info.Url,k,v)
			request.Header.Set(k,v)
		}
	}

	retryCount := 0
RETRY_LOOP:
	response, e := engine.client.Do(request)
	if e != nil {
		logger.WarnF("[Download] Retries: %d -> Url: \"%s\" Error: %v",retryCount,info.Url,e)
		retryCount += 1
		if retryCount >= engine.retries{
			e = fmt.Errorf("[Download] Retry \""+strconv.Itoa(engine.retries)+
				"\" but still can't Download Url: "+info.Url +" status: "+strconv.Itoa(response.StatusCode))
			result.E = e
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
				logger.WarnF("[Download] Retries: %d -> Url: \"%s\" Read %d bytes Error: %v", retryCount, info.Url, result.FileSize, e)
				retryCount += 1
				if retryCount >= engine.retries {
					e = fmt.Errorf("[Download] Retry \"" + strconv.Itoa(engine.retries) +
						"\" but still can't Read Url: " + info.Url + " -> already read %d bytes Data, status: " + strconv.Itoa(response.StatusCode))
					result.E = e
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
				result.E = e
				return result
			}
			logger.TraceF("[Download] Url:%s -> read %d bytes. Write %d bytes", info.Url, n,wn)
			result.FileSize += int64(wn)
		}
		logger.InfoF("[Download] 200 -> %s FileSize: %s", info.Url, util.GetFormatFileSize(result.FileSize))
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
				"\" but still can't Download Url: " + info.Url + " status: " + strconv.Itoa(response.StatusCode))
			result.E = e
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