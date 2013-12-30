package config

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "regexp"
  "sort"
  "strings"
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
type Config struct {
  Id                  int               `json:"id"`
  SourcePaths         StringSlice       `json:"source_path"`
  SourceParams        StringSlice       `json:"source_params"`
  TargetServer        string            `json:"target_server"`
  TargetPath          string            `json:"target_path"`
  TargetParamNameSwap map[string]string `json:"target_param_name_swap"`
  ConnectionTimeout   int               `json:"connection_timeout"`
  ResponseTimeout     int               `json:"response_timeout"`
  Redirect            bool              `json:"redirect"`
}

//总体配置文件结构
type ConfigFile struct {
  Default Config
  Configs []Config
  Listen  struct {
    Host string
    Port int
  }
  AccessLogFile string `json:"access_log_file"`
  ErrorLogFile  string `json:"error_log_file"`
  AdminUri      string `json:"admin_uri"`
  PprofHttpd    string `json:"pprof_httpd"`
}

//返回配置文件条目数
func (c *ConfigFile) Len() int {
  return len(c.Configs)
}

//=======================需要对 ConfigFile[].ID 进行排序
type sortById []Config

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
支持正则表达式
*/

func (cf *Config) FindBySourceParams(reqParams []string) (config *Config, err *ConfigErr) {
  var (
    vr *regexp.Regexp
    qp string
    sp string
  )
  for _, sp = range cf.SourceParams {
    if strings.HasPrefix(sp, "^") {
      vr = regexp.MustCompile(sp)
      for _, qp = range reqParams {
        if vr.MatchString(qp) {
          config = cf
          return
        }
      }
    } else {
      for _, qp = range reqParams {
        if qp == sp {
          config = cf
          return
        }
      }
    }
  }
  err = &ConfigErr{When: time.Now(), What: "no match by source params."}
  return
}

func (c *ConfigFile) FindBySourceParams(reqParams []string) (config *Config, err *ConfigErr) {
  for idx := 0; idx < len(c.Configs); idx++ {
    if config, err = c.Configs[idx].FindBySourceParams(reqParams); err == nil {
      return
    }
  }
  err = &ConfigErr{When: time.Now(), What: "no match by source params."}
  return
}

/**
根据请求路径查找
reqPath 当前请求的路径
配置文件中支持正则
*/
func (cf *Config) FindBySourcePath(reqPath string) (config *Config, err *ConfigErr) {
  for _, sp := range cf.SourcePaths {
    if strings.HasPrefix(sp, "^") && regexp.MustCompile(sp).MatchString(reqPath) {
      config = cf
      return
    } else if sp == reqPath {
      config = cf
      return
    }
  }
  err = &ConfigErr{When: time.Now(), What: "no match by source path."}
  return
}

func (c *ConfigFile) FindBySourcePath(reqPath string) (config *Config, err *ConfigErr) {
  //for 结构体,下标 要比 range 快好多,range 需要每个都重复赋值,而普通数据类型则无差别
  for idx := 0; idx < len(c.Configs); idx++ {
    if config, err = c.Configs[idx].FindBySourcePath(reqPath); err == nil {
      return
    }
  }

  err = &ConfigErr{When: time.Now(), What: "no match by source path."}
  return
}

//根据路径和参数进行查找,两个都匹配才返回对应配置
func (c *ConfigFile) FindBySourcePathAndParams(reqParams []string, reqPath string) (config *Config, err *ConfigErr) {
  var (
    pc, sc *Config
    pe, se *ConfigErr
  )
  for idx := 0; idx < len(c.Configs); idx++ {
    if pc, pe = c.Configs[idx].FindBySourcePath(reqPath); pe == nil {
      if sc, se = c.Configs[idx].FindBySourceParams(reqParams); se == nil && pc.Id == sc.Id {
        config = &c.Configs[idx]
        return
      }
    }
  }

  err = &ConfigErr{When: time.Now(), What: "no match by source path and source params."}
  return
}

//根据路径和参数查找,参数优先级比路径高,都找不到则返回默认值
func (c *ConfigFile) FindByParamsOrSourcePath(reqParams []string, reqPath string) (config *Config) {
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
func LoadConfigFile(fileName string) ConfigFile {
  b, err := ioutil.ReadFile(fileName)
  if err != nil {
    panic("Load Config File Error.")
  }
  return LoadConfig(b)
}

/**
解析配置文件
*/
func LoadConfig(b []byte) (cs ConfigFile) {
  if json.Unmarshal([]byte(b), &cs) != nil {
    panic("Parse json failed.")
  }
  sort.Sort(sortById(cs.Configs))
  if len(cs.Default.TargetServer) == 0 {
    panic("default target server must be configured.")
  }
  return
}
