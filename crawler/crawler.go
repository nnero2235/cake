package crawler

import (
	"cake/net"
	"cake/util/log"
	"strconv"
)

const linkBuffer int = 100

const tasksBuffer int = 100

var logger = log.GetLogger()

//how to process result is depend by user
type Processor interface {
	Process(url string,html string,linkChan chan<-string)
}

type Crawler struct {
	engine *net.NetEngine
	currentTasks int
	finishedTask int
	linkChan chan string
	tasksChan chan int
	processorPipeLine []*Processor
}

func CreateCrawler(processPipeLine []*Processor) *Crawler{
	if processPipeLine == nil || len(processPipeLine) == 0 {
		panic("Fatal Error: processList is nil")
	}
	return &Crawler{
		engine:       net.CreateEngine(),
		currentTasks: 0,
		linkChan:     make(chan string, linkBuffer),
		tasksChan:    make(chan int,tasksBuffer),
		processorPipeLine: processPipeLine,
	}
}

func (c *Crawler) Start(startUrl string) {
	logger.Trace("Crawler start...")
	c.currentTasks += 1
	go func(startUrl string) {
		if e := c.fetch(startUrl);e != nil {
			logger.Error(e.Error())
		}
	}(startUrl)
	c.listenOnLinkChan()
	logger.Trace("Crawler shutdown...")
}

func (c *Crawler) listenOnLinkChan(){
	logger.Trace("Listen on linkChan...")
LINK_LOOP:
	select {
	case link := <- c.linkChan:
		go func(){
			if e := c.fetch(link);e != nil {
				logger.Error(e.Error())
			}
		}()
		goto LINK_LOOP
	case <- c.tasksChan:
		c.currentTasks += 1
	}
}

func (c *Crawler) fetch(url string) error{
	if url == ""{
		logger.Warn("url is nil.")
		return nil
	}
	html, e := c.engine.Get(url)
	if e != nil{
		return e
	}
	for i,processor := range c.processorPipeLine {
		logger.Trace("Proccess_"+strconv.Itoa(i)+" doing...")
		(*processor).Process(url,html,c.linkChan)
	}
	return nil
}