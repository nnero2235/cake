package crawler

import (
	"cake/network"
	"cake/util"
	"cake/util/log"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	UrlChannelSize = 100
)

const (
	StatusSuccess = 1
	StatusFail = 2
)

var UserAgentList = []string{
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/534.57.2 (KHTML, like Gecko) Version/5.1.7 Safari/534.57.2",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.11 (KHTML, like Gecko) Chrome/23.0.1271.64 Safari/537.11",
		"Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US) AppleWebKit/534.16 (KHTML, like Gecko) Chrome/10.0.648.133 Safari/534.16",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/30.0.1599.101 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.1 (KHTML, like Gecko) Chrome/21.0.1180.71 Safari/537.1 LBBROWSER",
		"Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; WOW64; Trident/5.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; Media Center PC 6.0; .NET4.0C; .NET4.0E; LBBROWSER)",
		"Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1; SV1; QQDownload 732; .NET4.0C; .NET4.0E; LBBROWSER)",
		"Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; WOW64; Trident/5.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; Media Center PC 6.0; .NET4.0C; .NET4.0E; QQBrowser/7.0.3698.400)",
		"Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1; SV1; QQDownload 732; .NET4.0C; .NET4.0E)",
		"Mozilla/5.0 (Windows NT 5.1) AppleWebKit/535.11 (KHTML, like Gecko) Chrome/17.0.963.84 Safari/535.11 SE 2.X MetaSr 1.0",
		"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 5.1; Trident/4.0; SV1; QQDownload 732; .NET4.0C; .NET4.0E; SE 2.X MetaSr 1.0)",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/38.0.2125.122 UBrowser/4.0.3214.0 Safari/537.36",
}

var logger = log.GetLogger()

//interface of processor: for business obj process result and provide links
type Processor interface {
	//process result and return links
	Process(link *Link,html string) []*Link
}

//for url duplicate filter
type URLFilter interface {
	//check duplicate url
	CheckDuplicate(url string) bool
}

//Link struct contains url and attachAttrMap
type Link struct {
	Url string
	AttrMap map[string]string
}

//crawler controller: schedule go route to fetch web page
type Crawler struct {
	startLink           string              //where to start
	processor           Processor           //business processor
	urlFilter           URLFilter           //business impl url filter
	httpClient          *network.HttpEngine //for fetch url pages
	shutdown            bool                //for stop crawler or finished close
	urlChannel          chan *Link          //url queue to fetch
	resultChannel       chan int            //status of result
	WaitTime            time.Duration       //fetch Interval time
	CurrentFetchedPages int                 //current fetched pages statistics
	CurrentFailPages    int                 //current fail pages statistics
	wg                  *sync.WaitGroup     //wait for shutdown
}

func randomUserAgent() string {
	return UserAgentList[rand.Intn(len(UserAgentList))]
}

func UseUserAgent(index int) string {
	return UserAgentList[index]
}

func crawlerValid(c *Crawler) error {
	if c == nil {
		return fmt.Errorf("crawler is nil. Nothing to Do")
	}
	if c.startLink == "" {
		return fmt.Errorf("startLink must be specified. Nothing to Do")
	}
	if c.processor == nil {
		return fmt.Errorf("processor is nil. Nothing to Do")
	}
	if c.urlFilter == nil {
		return fmt.Errorf("urlFilter is nil. Nothing to Do")
	}
	if c.httpClient == nil {
		return fmt.Errorf("httpClient is nil. Nothing to Do")
	}
	return nil
}

func CreateCrawler(processor Processor,filter URLFilter,startLink string) *Crawler {
	return &Crawler{
		startLink:     startLink,
		processor:     processor,
		urlFilter:     filter,
		httpClient:    network.CreateEngine(),
		shutdown:      false,
		urlChannel:    make(chan *Link,UrlChannelSize),
		resultChannel: make(chan int,UrlChannelSize),
		WaitTime:      util.GetTimeMilliSecond(500),
		wg:            &sync.WaitGroup{},
	}
}

func CreateCrawlerByMaxThreads(maxThreads int,processor Processor,filter URLFilter,startLink string) *Crawler {
	return &Crawler{
		startLink:     startLink,
		processor:     processor,
		urlFilter:     filter,
		httpClient:    network.CreateEngineByParams(maxThreads,network.Timeout,network.Retries),
		shutdown:      false,
		urlChannel:    make(chan *Link,UrlChannelSize),
		resultChannel: make(chan int,UrlChannelSize),
		WaitTime:      util.GetTimeMilliSecond(500),
		wg:            &sync.WaitGroup{},
	}
}

//main method to start fetch web pages
//blocking util finish
//fetch concurrent inner
func (c *Crawler) Start() {
	e := crawlerValid(c)
	if e != nil {
		logger.ErrorF("%v",e)
		return
	}
	//statistic time cost
	defer util.FuncElapsed("Crawler Start")()

	//add first url
	c.urlChannel <- &Link{
		Url:     c.startLink,
		AttrMap: nil,
	}
	//fetch loop
	for !c.shutdown {
		select {
		case link,ok := <- c.urlChannel:
			if !ok {
				c.shutdown = true
				continue
			}
			if !c.urlFilter.CheckDuplicate(link.Url) {
				c.wg.Add(1)
				go c.fetch(link) //concurrent fetch
			} else {
				logger.TraceF("url: %s duplicate. Skip!",link.Url)
			}
		case status := <- c.resultChannel:
			if status == StatusSuccess {
				c.CurrentFetchedPages += 1
			} else {
				c.CurrentFailPages += 1
			}
		case <- time.After(util.GetTimeSecond(15)): //timeout for select: we determine this for shutdown
			close(c.urlChannel)
		}
	}
	c.wg.Wait() //wait for all go route done
	logger.Info("Crawler Shutdown.")
}


func (c *Crawler) fetch(link *Link) {
	defer c.wg.Done()
	headerMap := make(map[string]string)
	headerMap["UserAgent"] = UseUserAgent(0)
	html, e := c.httpClient.Get(link.Url, headerMap)
	if e != nil {
		logger.ErrorF("%v",e)
		c.resultChannel <- StatusFail
		return
	}
	//process result
	links := c.processor.Process(link,html)
	//send link to fetch
	for _,link := range links {
		c.urlChannel <- link
	}
	time.Sleep(c.WaitTime)
	c.resultChannel <- StatusSuccess
}
