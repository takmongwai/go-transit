package config

import (
  "encoding/json"
  "io/ioutil"
  "sort"
  "time"
)

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

//根据请求源参数在整个文件中进行查询
func (c *ConfigFile) FindBySourceParams(reqParams []string) (config *Config, err *ConfigErr) {
  for idx := 0; idx < len(c.Configs); idx++ {
    if config, err = c.Configs[idx].FindBySourceParams(reqParams); err == nil {
      return
    }
  }
  err = &ConfigErr{When: time.Now(), What: "no match by source params."}
  return
}

//根据请求源路径在整个文件中进行查询
func (c *ConfigFile) FindBySourcePath(reqPath string) (config *Config, err *ConfigErr) {
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
