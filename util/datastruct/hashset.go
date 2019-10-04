package datastruct

import (
	"cake/util/log"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
)

const (
	PoolSize = 128
)

var logger = log.GetLogger()
var defaultNameNumber int32 = 1 //for default name
var globalObj = &struct {}{} //flag to save memory in map's value

//thread safe pool for url save
type HashSet struct{
	Name string
	innerMap map[interface{}]*struct{} //set impl by map
	bufferChan chan interface{} //for concurrent use
	closed bool //for concurrent use
}

func CreateHashSet(name string) *HashSet {
	if name == "" {
		number := atomic.AddInt32(&defaultNameNumber,1)
		name = "Default-Set-"+strconv.Itoa(int(number))
		logger.Warn("No name specified.Using name \"" + name +"\"")
	}
	return &HashSet{
		Name:   name,
		innerMap: make(map[interface{}]*struct{},PoolSize),
	}
}

//this is for concurrent use
//using this will open a listen go route to add async
func CreatConcurrentHashSet(name string,bufferSize int) *HashSet {
	s := CreateHashSet(name)
	s.bufferChan = make(chan interface{},bufferSize)
	go func() {
		for{
			select {
			case v,ok := <- s.bufferChan:
				if !ok { //chan closed
					logger.Debug("ConcurrentHashSet is Closed.")
					return
				}
				switch channel := v.(type) {
				case chan interface{}:
					value := <- channel
					logger.DebugF("Add v: %v",value)
					success := s.Add(value)
					channel <- success
				default: //means remove
					logger.DebugF("Remove v: %v",v)
					s.Remove(v)
				}
			}
		}
	}()
	return s
}

//add value to set.
//return true for success, false for duplicate
//not thread safe
func (p *HashSet) Add(val interface{}) bool {
	if val == nil{
		logger.Warn("val is nil.Do not add to \""+p.Name+"\".")
		return false
	}
	if _,ok := p.innerMap[val]; !ok {
		p.innerMap[val] = globalObj
		return true
	}
	return false
}

//remove value from set
//not thread safe
//always true for remove
func (p *HashSet) Remove(val interface{}) bool{
	if val == nil{
		logger.Warn("val is nil.Remove nothing in \""+p.Name+"\".")
		return false
	}
	delete(p.innerMap,val)
	return true
}

func (p *HashSet) Size() int{
	return len(p.innerMap)
}

//add async to thread safe
func (p *HashSet) AddConcurrent(val interface{}) bool{
	if p.bufferChan == nil {
		logger.Error("Please using \"CreateHashSetConcurrent()\" for concurrent use.")
		return false
	}
	if p.closed {
		logger.Warn("ConcurrentHashSet was Closed.Nothing was Added")
		return false
	}
	wrapper := make(chan interface{})
	p.bufferChan <- wrapper
	wrapper <- val
	success := <- wrapper
	switch s := success.(type) {
	case bool:
		return s
	default:
		logger.Error("Unknown type for Add result.")
		return false
	}
}

func (p *HashSet) RemoveConcurrent(val interface{}) {
	if p.bufferChan == nil {
		logger.Error("Please using \"CreateHashSetConcurrent()\" for concurrent use.")
		return
	}
	if p.closed {
		logger.Warn("ConcurrentHashSet was Closed.Nothing was Removed")
		return
	}
	p.bufferChan <- val
}

func (p *HashSet) Close(){
	if p.bufferChan != nil {
		//must be thread safe
		m := sync.Mutex{}
		defer m.Unlock()
		m.Lock()
		p.closed = true
		close(p.bufferChan)
	}
}

//just for debug
//using this in production,may cause unexpect errors
func (p *HashSet) PrintSet(){
	if p.Size() == 0 {
		logger.Info("HashSet: \""+p.Name+"\" has no elements.")
		return
	}
	resultStr := "HashSet: \""+p.Name+"\" values (\n"
	for k := range p.innerMap{
		kName := ""
		switch str := k.(type) {
		case string:
			kName = str
		case fmt.Stringer:
			kName = str.String()
		default:
			logger.Error("Value in HashSet do not impl \"String\" type. Can't Print.")
			continue
		}
		resultStr += "\t"+kName+"\n"
	}
	resultStr += ")"
	logger.Info(resultStr)
}