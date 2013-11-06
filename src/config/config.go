package config

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "sort"
  "time"
)

type ConfigErr struct {
  When time.Time
  What string
}

func (e ConfigErr) Error() string {
  return fmt.Sprintf("%v: %v", e.When, e.What)
}

type StringSlice []string

//配置项
type ConfigT struct {
  Id                  int               `json:"id"`
  SourcePaths         StringSlice       `json:"source_path"`
  SourceParams        StringSlice       `json:"source_params"`
  TargetServer        string            `json:"target_server"`
  TargetPath          string            `json:"target_path"`
  TargetParamNameSwap map[string]string `json:"target_param_name_swap"`
  ConnectionTimeout   int               `json:"connection_timeout"`
  ResponseTimeout     int               `json:"response_timeout"`
  LogFile             string            `json:"log_file"`
}

//总体配置文件结构
type ConfigFileT struct {
  Default ConfigT
  Configs []ConfigT
  Listen  struct {
    Host string
    Port int
  }
}

//返回配置文件条目数
func (c *ConfigFileT) Len() int {
  return len(c.Configs)
}

//=======================需要对 ConfigFile[].ID 进行排序
type sortById []ConfigT

func (v sortById) Len() int           { return len(v) }
func (v sortById) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v sortById) Less(i, j int) bool { return v[i].Id < v[j].Id }

/**
如果一个字符串数组中包含指定字符串,则返回该字符串所在下标
*/
func (ss StringSlice) Pos(s string) int {
  for p, v := range ss {
    if v == s {
      return p
    }
  }
  return -1
}

/**
如两个字符串数组中有相同元素,则返回true
*/
func (ss StringSlice) IsInclude(s StringSlice) bool {
  for _, v := range ss {
    for _, v1 := range s {
      if v == v1 {
        return true
      }
    }
  }
  return false
}

/**
根据参数查询
reqParams 传入的参数数组,类似 p=1 p1=pp 等参数对
*/
func (c *ConfigFileT) FindBySourceParams(reqParams []string) (config *ConfigT, err *ConfigErr) {
  for _, cf := range c.Configs {
    if cf.SourceParams.IsInclude(reqParams) {
      config = &cf
      return
    }
  }
  err = &ConfigErr{When: time.Now(), What: "no match by source params."}
  return
}

/**
根据请求路径查找
reqPath 当前请求的路径
*/
func (c *ConfigFileT) FindBySourcePath(reqPath string) (config *ConfigT, err *ConfigErr) {
  for _, cf := range c.Configs {
    if cf.SourcePaths.Pos(reqPath) != -1 {
      config = &cf
      return
    }
  }
  err = &ConfigErr{When: time.Now(), What: "no match by source path."}
  return
}

//根据路径和参数查找,参数优先级比路径高,都找不到则返回默认值
func (c *ConfigFileT) FindByBoth(reqParams []string, reqPath string) (config *ConfigT) {
  var err *ConfigErr
  if config, err = c.FindBySourceParams(reqParams); err != nil {
    if config, err = c.FindBySourcePath(reqPath); err != nil {
      config = &c.Default
    }
  }
  return
}

/**
读取配置文件
*/
func LoadConfigFile(fileName string) ConfigFileT {
  b, err := ioutil.ReadFile(fileName)
  if err != nil {
    panic("Load Config File Error.")
  }
  return LoadConfig(b)
}

/**
解析配置文件
*/
func LoadConfig(b []byte) (cs ConfigFileT) {
  if json.Unmarshal([]byte(b), &cs) != nil {
    panic("Parse json failed.")
  }
  sort.Sort(sortById(cs.Configs))
  return
}

/*
日期格式转换,将 %Y %m %d 等转换成 golang 的样式
*/
func time_format(){
  
}
