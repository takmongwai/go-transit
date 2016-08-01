package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

import _ "net/http/pprof"

type RuntimeEnv struct {
	FullPath  string
	Home      string
	AccessLog *log.Logger
	ErrorLog  *log.Logger
}

var globalConfig ConfigFile
var globalEnv RuntimeEnv
var accessUserMap *AccessUserMap

func init() {
	var (
		fullpath string
		err      error
	)
	if fullpath, err = filepath.Abs(os.Args[0]); err != nil {
		log.Fatal(err)
	}
	globalEnv.FullPath = fullpath
	if strings.HasSuffix(filepath.Dir(fullpath), "bin") {
		fp, _ := filepath.Abs(filepath.Join(filepath.Dir(fullpath), ".."))
		globalEnv.Home = fp
	} else {
		globalEnv.Home = filepath.Dir(fullpath)
	}

	accessUserMap = NewAccessUserMap()
}

func fileExists(name string) bool {

	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true

}

func showUsage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s \n",
		os.Args[0])
	flag.PrintDefaults()
}

func findConfigFile() (cf string, err error) {
	tryFiles := []string{
		filepath.Join(globalEnv.Home, "etc", "config.json"),
	}

	for _, cf = range tryFiles {
		if fileExists(cf) {
			log.Printf("INFO: Check config file %s", cf)
			return
		}
	}

	err = fmt.Errorf("ERROR: Can't find any config file.")
	return
}

func initDir(dir string) {
	if !fileExists(dir) {
		os.MkdirAll(dir, 0755)
	}
}

// initAccessUser 读取配置文件初始化访问用户列表
func initAccessUser(fileName string) error {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	aus := make([]*AccessUser, 0)
	err = json.Unmarshal(b, &aus)
	if err != nil {
		return err
	}
	for _, au := range aus {
		accessUserMap.Put(au.ID, au)
	}
	return nil
}

// initAccessLog 初始化访问日志,如果配置文件中该属性留空,则将访问日志输出到 stdout 标准输出中
func initAccessLog() {
	logPath := globalConfig.AccessLog

	if len(strings.TrimSpace(logPath)) == 0 {
		globalEnv.AccessLog = log.New(os.Stdout, "", log.LstdFlags)
		return
	}

	if filepath.IsAbs(logPath) {
		globalEnv.AccessLog = fileLogger(logPath)
		return
	}

	if fap, err := filepath.Abs(filepath.Join(globalEnv.Home, globalConfig.AccessLog)); err == nil {
		globalEnv.AccessLog = fileLogger(fap)
		return
	}

}

// initErrorLog 初始化错误日志,如果配置文件中该属性留空,则将错误日志输出到 stderr 标准输出中
func initErrorLog() {
	logPath := globalConfig.ErrorLog

	if len(strings.TrimSpace(logPath)) == 0 {
		globalEnv.ErrorLog = log.New(os.Stdout, "", log.LstdFlags)
		return
	}

	if filepath.IsAbs(logPath) {
		globalEnv.ErrorLog = fileLogger(logPath)
		return
	}

	if fap, err := filepath.Abs(filepath.Join(globalEnv.Home, logPath)); err == nil {
		globalEnv.ErrorLog = fileLogger(fap)
		return
	}

}

func fileLogger(logPath string) (logger *log.Logger) {
	if !filepath.IsAbs(logPath) {
		return
	}
	initDir(filepath.Dir(logPath))
	if out, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModeAppend|0666); err == nil {
		logger = log.New(out, "", 0)
		logger.Printf("#start at: %s\n", time.Now())
	} else {
		log.Fatal(err)
	}
	return
}

func main() {
	var (
		err            error
		configFile     string
		accessUserFile string
		host           string
		port           int
	)

	flag.Usage = showUsage
	flag.StringVar(&configFile, "f", "", "config file path")
	flag.StringVar(&accessUserFile, "u", "", "access users file path")
	flag.IntVar(&port, "p", 9000, "listen port,default 9000")
	flag.StringVar(&host, "h", "", "listen ip,default 127.0.0.1")
	flag.Parse()

	if len(configFile) == 0 {
		configFile, err = findConfigFile()
		if err != nil {
			log.Fatal(err)
		}
	}

	if !fileExists(configFile) {
		log.Fatal("ERROR: Can't find any config file.")
		os.Exit(1)
	}

	log.Printf(`INFO: Using config file "%s"`, configFile)
	globalConfig = LoadConfigFile(configFile)

	if globalConfig.Listen.Port <= 0 {
		globalConfig.Listen.Port = port
	}
	if len(host) != 0 {
		globalConfig.Listen.Host = host
	}
	initAccessLog()
	initErrorLog()
	if err := initAccessUser(accessUserFile); err != nil {
		log.Fatal(err)
	}
	fmt.Println("pprof listen on:", globalConfig.PprofHttpd)
	if len(globalConfig.PprofHttpd) != 0 {
		go func() {
			log.Println(http.ListenAndServe(globalConfig.PprofHttpd, nil))
		}()
	}
	Run()
}
