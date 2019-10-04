package log

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DEBUG = iota
	TRACE
	INFO
	WARN
	ERROR
)

type LogConfig struct{
	Name string
	FilePath string
	FileName string
	ConsoleOutput bool
	Level string
	LevelValue int
}

type Logger struct {
	config *LogConfig
	consoleOutput *log.Logger
	debugOutput *log.Logger
	traceOutput *log.Logger
	infoOutput *log.Logger
	warnOutput *log.Logger
	errorOutput *log.Logger
}

//init when use this package
func init(){
	//double check for thread safe
	if !inited {
		mutex := sync.Mutex{}
		defer mutex.Unlock()
		mutex.Lock()
		if !inited {
			InitLoggers(".\\") //default path is current dir
		}
	}
}

//initLoggers can be invoked only once
var inited = false

var rootLogger *Logger

//specially panic log package unexpect error
func panicError(format string,v... interface{}){
	str := fmt.Sprintf(format+"\n",v)
	panic(str)
}

func createGlobalLogger() *Logger{
	config := &LogConfig{
		Name:          "Global",
		FilePath:      "D:\\tmp\\go\\cake",
		FileName:      "cake",
		ConsoleOutput: true,
		Level:         "INFO",
		LevelValue:    INFO,
	}
	logger, e := createLogger(config)
	if e != nil {
		fmt.Printf("Fatal Error: %v",e)
		os.Exit(0)
	}
	return logger
}

func createFromConfigFile(fileRelativeName string) (*LogConfig,error){
	if fileRelativeName == "" {
		return nil,fmt.Errorf("Log Config file full Name need.")
	}
	configFile, e := os.Open(fileRelativeName)
	if e != nil{
		return nil,fmt.Errorf("Config File:%s can't be opened. Error:%v", fileRelativeName,e)
	}
	content, e := ioutil.ReadAll(configFile)
	if e != nil{
		return nil,fmt.Errorf("Read Config File Error:%v",e)
	}
	config := &LogConfig{}
	e = json.Unmarshal(content, config)
	if e != nil{
		return nil,fmt.Errorf("Config File Parse Error:%v",e)
	}
	if config.Level == "DEBUG" {
		config.LevelValue = DEBUG
	} else if config.Level == "TRACE" {
		config.LevelValue = TRACE
	} else if config.Level == "INFO" {
		config.LevelValue = INFO
	} else if config.Level == "WARN" {
		config.LevelValue = WARN
	} else if config.Level == "ERROR" {
		config.LevelValue = ERROR
	} else {
		return nil,fmt.Errorf("Fatal Error: unknow Log Level: %s",config.Level)
	}
	return config,nil
}

func createLoggerFile(filePath string,fileName string,level string) (*os.File,error){
	if _, e := os.Open(filePath); os.IsNotExist(e){
		if e := os.MkdirAll(filePath, os.ModePerm);e != nil{
			return nil,fmt.Errorf("Fatal Error: %v",e)
		}
		fmt.Println("log file Path doesn't exists.Create it!")
	}
	today := time.Now().Format("2006-01-02")
	logWriter, e := os.OpenFile(filePath+"\\"+fileName+today+"_"+level+".log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if e != nil {
		return nil,fmt.Errorf("Fatal Error: %v",e)
	}
	return logWriter,nil
}

func createLogger(config *LogConfig) (*Logger,error){
	if config == nil{
		return nil,fmt.Errorf("Fatal Error: config is nil.")
	}
	if config.FileName == "" { //just console
		output := log.New(os.Stdout,"",log.Ldate | log.Ltime | log.Lshortfile)
		return &Logger{config:config, consoleOutput:output,},nil
	} else {
		logger := &Logger{config: config}
		if config.ConsoleOutput {
			logger.consoleOutput = log.New(os.Stdout,"",log.Ldate | log.Ltime | log.Lshortfile)
		}
		if config.LevelValue <= DEBUG {
			fileWriter,e := createLoggerFile(config.FilePath, config.FileName, "DEBUG")
			if e != nil {
				return nil,e
			}
			logger.debugOutput = log.New(fileWriter,"",log.Ldate | log.Ltime | log.Lshortfile)
		}
		if config.LevelValue <= TRACE {
			fileWriter,e := createLoggerFile(config.FilePath, config.FileName, "TRACE")
			if e != nil {
				return nil,e
			}
			logger.traceOutput = log.New(fileWriter,"",log.Ldate | log.Ltime | log.Lshortfile)
		}
		if config.LevelValue <= INFO {
			fileWriter,e := createLoggerFile(config.FilePath, config.FileName, "INFO")
			if e != nil {
				return nil,e
			}
			logger.infoOutput = log.New(fileWriter,"",log.Ldate | log.Ltime | log.Lshortfile)
		}
		if config.LevelValue <= WARN {
			fileWriter,e := createLoggerFile(config.FilePath, config.FileName, "WARN")
			if e != nil {
				return nil,e
			}
			logger.warnOutput = log.New(fileWriter,"",log.Ldate | log.Ltime | log.Lshortfile)
		}
		if config.LevelValue <= ERROR {
			fileWriter,e := createLoggerFile(config.FilePath, config.FileName, "ERROR")
			if e != nil {
				return nil,e
			}
			logger.errorOutput = log.New(fileWriter,"",log.Ldate | log.Ltime | log.Lshortfile)
		}
		return logger,nil
	}
}

//init logger by default logic
//scan path get format json config file.
// file name is format. xxx_log.json
func InitLoggers(configFilePath string){
	fileInfos, e := ioutil.ReadDir(configFilePath)
	if e != nil {
		if os.IsNotExist(e){
			fmt.Printf("Init Loggers fail: path \"%s\" does not exists.",configFilePath)
			return
		}
		panicError("Fatal Error: %v",e)
	}
	if len(fileInfos) == 0 {
		fmt.Printf("Init Loggers fail: there is no file found in path \"%s\"",configFilePath)
		return
	}
	for _,file := range fileInfos{
		if !file.IsDir(){
			name := file.Name()
			if strings.HasSuffix(name,"_log.json") {
				fmt.Printf("Logger Found config file: %s\n",name)
				config, e := createFromConfigFile(configFilePath+"\\"+name)
				if e != nil {
					panicError("%v",e)
				}
				logger,e := createLogger(config)
				if e != nil {
					panicError("%v",e)
				}
				rootLogger = logger
				fmt.Printf("Logger Config File: %s loaded\n",name)
				fmt.Printf("Init Loggers successful.\n")
				inited = true
				return //only one file can be loaded
			}
		}
	}
}

//Get Logger which from config file
//if there is no config file, just create and using the default one
func GetLogger() *Logger {
	if rootLogger == nil {
		mutex := sync.Mutex{}
		defer mutex.Unlock()
		mutex.Lock()
		if rootLogger == nil {
			fmt.Printf("name: \"configed logger\" not found. create and using global logger!\n")
			rootLogger = createGlobalLogger()
		}
	}
	return rootLogger
}

func (logger *Logger) Debug(msg string){
	if logger.config.LevelValue <= DEBUG {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[DEBUG] %s",msg)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[DEBUG] %s",msg)
		}
	}
}

func (logger *Logger) DebugF(format string,values... interface{}){
	if logger.config.LevelValue <= DEBUG {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[DEBUG] "+format,values...)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[DEBUG] "+format,values...)
		}
	}
}

func (logger *Logger) Trace(msg string){
	if logger.config.LevelValue <= TRACE {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[TRACE] %s",msg)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[TRACE] %s",msg)
		}
		if logger.traceOutput != nil {
			logger.traceOutput.Printf("[TRACE] %s",msg)
		}
	}
}

func (logger *Logger) TraceF(format string,values... interface{}){
	if logger.config.LevelValue <= TRACE {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[TRACE] "+format,values...)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[TRACE] "+format,values...)
		}
		if logger.traceOutput != nil {
			logger.traceOutput.Printf("[TRACE] "+format,values...)
		}
	}
}

func (logger *Logger) Info(msg string){
	if logger.config.LevelValue <= INFO {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[INFO] %s",msg)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[INFO] %s",msg)
		}
		if logger.traceOutput != nil {
			logger.traceOutput.Printf("[INFO] %s",msg)
		}
		if logger.infoOutput != nil {
			logger.infoOutput.Printf("[INFO] %s",msg)
		}
	}
}

func (logger *Logger) InfoF(format string,values... interface{}){
	if logger.config.LevelValue <= INFO {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[INFO] "+format,values...)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[INFO] "+format,values...)
		}
		if logger.traceOutput != nil {
			logger.traceOutput.Printf("[INFO] "+format,values...)
		}
		if logger.infoOutput != nil {
			logger.infoOutput.Printf("[INFO] "+format,values...)
		}
	}
}

func (logger *Logger) Warn(msg string){
	if logger.config.LevelValue <= WARN {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[WARN] %s",msg)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[WARN] %s",msg)
		}
		if logger.traceOutput != nil {
			logger.traceOutput.Printf("[WARN] %s",msg)
		}
		if logger.infoOutput != nil {
			logger.infoOutput.Printf("[WARN] %s",msg)
		}
		if logger.warnOutput != nil {
			logger.warnOutput.Printf("[WARN] %s",msg)
		}
	}
}

func (logger *Logger) WarnF(format string,values ...interface{}){
	if logger.config.LevelValue <= WARN {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[WARN] "+format,values...)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[WARN] "+format,values...)
		}
		if logger.traceOutput != nil {
			logger.traceOutput.Printf("[WARN] "+format,values...)
		}
		if logger.infoOutput != nil {
			logger.infoOutput.Printf("[WARN] "+format,values...)
		}
		if logger.warnOutput != nil {
			logger.warnOutput.Printf("[WARN] "+format,values...)
		}
	}
}

func (logger *Logger) Error(msg string){
	if logger.config.LevelValue <= ERROR {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[ERROR] %s",msg)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[ERROR] %s",msg)
		}
		if logger.traceOutput != nil {
			logger.traceOutput.Printf("[ERROR] %s",msg)
		}
		if logger.infoOutput != nil {
			logger.infoOutput.Printf("[ERROR] %s",msg)
		}
		if logger.warnOutput != nil {
			logger.warnOutput.Printf("[ERROR] %s",msg)
		}
		if logger.errorOutput != nil {
			logger.errorOutput.Printf("[ERROR] %s",msg)
		}
	}
}

func (logger *Logger) ErrorF(format string,values... interface{}){
	if logger.config.LevelValue <= ERROR {
		if logger.consoleOutput != nil {
			logger.consoleOutput.Printf("[ERROR] "+format,values...)
		}
		if logger.debugOutput != nil {
			logger.debugOutput.Printf("[ERROR] "+format,values...)
		}
		if logger.traceOutput != nil {
			logger.traceOutput.Printf("[ERROR] "+format,values...)
		}
		if logger.infoOutput != nil {
			logger.infoOutput.Printf("[ERROR] "+format,values...)
		}
		if logger.warnOutput != nil {
			logger.warnOutput.Printf("[ERROR] "+format,values...)
		}
		if logger.errorOutput != nil {
			logger.errorOutput.Printf("[ERROR] "+format,values...)
		}
	}
}