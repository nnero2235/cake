package datastruct

import (
	"sync"
	"testing"
)

func TestHashSet(t *testing.T){
	s := CreateHashSet("UrlPool-1")
	success := s.Add("https://www.baidu.com")
	if !success {
		t.Errorf("adding fail1")
	}
	s.PrintSet()
	success = s.Add("https://www.oschina.com")
	if !success {
		t.Errorf("adding fail2")
	}
	s.PrintSet()
	success = s.Add("https://www.baidu.com")
	if success {
		t.Errorf("duplicate data,but success")
	}
	s.PrintSet()
	s.Remove("https://www.baidu.com")
	if s.Size() != 1 {
		t.Errorf("remove fail")
	}
	s.PrintSet()
}

func TestConcurrentHashSet(t *testing.T){
	s := CreatConcurrentHashSet("UrlPool-2",5)
	wg := sync.WaitGroup{}
	wg.Add(4)
	go func() {
		success := s.AddConcurrent("https://www.baidu.com")
		if !success {
			t.Errorf("adding 1 www.baidu.com")
		}
		wg.Done()
	}()
	go func() {
		success := s.AddConcurrent("https://www.oschina.com")
		if !success {
			t.Errorf("adding 2 www.oschina.com")
		}
		wg.Done()
	}()
	go func() {
		success := s.AddConcurrent("https://www.baidu.com")
		if !success {
			t.Errorf("adding 3 www.baidu.com")
		}
		wg.Done()
	}()
	go func() {
		s.RemoveConcurrent("https://www.baidu.com")
		wg.Done()
	}()
	wg.Wait()
	s.PrintSet()
	s.Close()
}
