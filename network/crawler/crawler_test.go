package crawler

import (
	"bytes"
	"cake/util/datastruct"
	"github.com/PuerkitoBio/goquery"
	"testing"
)

type cnblogs struct {
	startLink string
	urlPool *datastruct.HashSet
}

func NewCNBlogs() *cnblogs {
	return &cnblogs{
		startLink: "https://www.cnblogs.com/",
		urlPool:   datastruct.CreateHashSet("cnblogs-1"),
	}
}

func (w *cnblogs) Process(link *Link,html string) []*Link {
	reader := bytes.NewReader([]byte(html))
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		logger.ErrorF("%v",err)
		return nil
	}
	doc.Find("div#main div#post_list div.post_item_body").Each(func(i int, s *goquery.Selection) {
		title := s.Find("a.titlelnk").Text()
		view := s.Find("span.article_view").Text()
		logger.InfoF("Title: %s -> \"%s\"",title,view)
	})
	var links []*Link
	doc.Find("div#main div#pager_bottom div.pager a").Each(func(i int, s *goquery.Selection) {
		link, exists := s.Attr("href")
		if exists {
			links = append(links,&Link{
				Url:     "https://www.cnblogs.com"+link,
				AttrMap: nil,
			})
		}
	})
	return links
}

func (w *cnblogs) CheckDuplicate(url string) bool {
	success := w.urlPool.Add(url)
	return !success
}

func TestCrawler_Start(t *testing.T) {
	cb := NewCNBlogs()
	c := CreateCrawler(cb,cb,cb.startLink)
	c.Start()
}