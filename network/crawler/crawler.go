package crawler

import (
	"cake/network"
	"cake/util"
	"cake/util/log"
	"fmt"
	"math/rand"
	"time"
)

const (
	UrlChannelSize = 100
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
	Process(html string) []string
}

//for url duplicate filter
type URLFilter interface {
	//check duplicate url
	CheckDuplicate(url string) bool
}

//crawler controller: schedule go route to fetch web page
type Crawler struct {
	startLink string //where to start
	processor Processor //business processor
	urlFilter URLFilter //business impl url filter
	httpClient *network.HttpEngine //for fetch url pages
	shutdown bool //for stop crawler or finished close
	urlChannel chan string //url queue to fetch
	timerChannel <- chan time.Time //timer ticker
	waitTime time.Duration //fetch Interval time
}

func randomUserAgent() string {
	return UserAgentList[rand.Intn(len(UserAgentList))]
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
		startLink:  startLink,
		processor:  processor,
		urlFilter:  filter,
		httpClient: network.CreateEngine(),
		shutdown:   false,
		urlChannel: make(chan string,UrlChannelSize),
		timerChannel: time.Tick(util.GetTimeSecond(3)),
		waitTime:    util.GetTimeMilliSecond(500),
	}
}

func CreateCrawlerByMaxThreads(maxThreads int,processor Processor,filter URLFilter,startLink string) *Crawler {
	return &Crawler{
		startLink:  startLink,
		processor:  processor,
		urlFilter:  filter,
		httpClient: network.CreateEngineByParams(maxThreads,network.Timeout,network.Retries),
		shutdown:   false,
		urlChannel: make(chan string,UrlChannelSize),
		timerChannel: time.Tick(util.GetTimeSecond(3)),
		waitTime:    util.GetTimeMilliSecond(500),
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

	activeTime := time.Now()
	//add first url
	c.urlChannel <- c.startLink
	//fetch loop
	for !c.shutdown {
		select {
		case url,ok := <- c.urlChannel:
			if !ok {
				c.shutdown = true
				continue
			}
			activeTime = time.Now()
			if !c.urlFilter.CheckDuplicate(url) {
				go c.fetch(url) //concurrent fetch
			} else {
				logger.TraceF("url: %s duplicate. Skip!",url)
			}
		case <- c.timerChannel:
			idleTime := time.Since(activeTime)
			if idleTime >= util.GetTimeSecond(15) { //should shutdown
				logger.InfoF("Idle Time: %v  over time. Shutdown!",idleTime)
				close(c.urlChannel)
			}
		}
	}
	logger.Info("Crawler Finished.")
}


func (c *Crawler) fetch(url string) {
	headerMap := make(map[string]string)
	headerMap["UserAgent"] = randomUserAgent()
	html, e := c.httpClient.Get(url, headerMap)
	if e != nil {
		logger.ErrorF("%v",e)
		return
	}
	//process result
	links := c.processor.Process(html)
	//send link to fetch
	for _,link := range links {
		c.urlChannel <- link
	}
	time.Sleep(c.waitTime)
}
