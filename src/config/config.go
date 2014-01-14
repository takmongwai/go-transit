package config

import (
  "fmt"
  "regexp"
  "strconv"
  "strings"
  "sync"
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

type cacheMap map[string]*Config

var cacheLock = sync.Mutex{}

var (
  paramCache = make(cacheMap)
  pathCache  = make(cacheMap)
)

func (cc cacheMap) set(key string, val *Config) {
  cacheLock.Lock()
  defer cacheLock.Unlock()
  cc[key] = val
}

func (cc cacheMap) get(key string) (val *Config, ok bool) {
  val, ok = cc[key]
  return
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

  for _, qp = range reqParams {
    if cf, ok := paramCache.get(qp); ok {
      config = cf
      return
    }
  }

  for _, sp = range cf.SourceParams {
    if strings.HasPrefix(sp, "^") {
      vr = regexp.MustCompile(sp)
      for _, qp = range reqParams {
        if vr.MatchString(qp) {
          config = cf
          paramCache.set(qp, cf)
          return
        }
      }
      continue
    } else {
      for _, qp = range reqParams {
        if qp == sp {
          config = cf
          paramCache.set(qp, cf)
          return
        }
      }
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
  var cache_key string

  cache_key = strconv.Itoa(cf.Id) + reqPath
  if cfg, ok := pathCache.get(cache_key); ok {
    config = cfg
    return
  }

  for _, sp := range cf.SourcePaths {
    cache_key = strconv.Itoa(cf.Id) + reqPath
    if strings.HasPrefix(sp, "^") && regexp.MustCompile(sp).MatchString(reqPath) {
      config = cf
      pathCache.set(cache_key, cf)
      return
    } else if sp == reqPath {
      config = cf
      pathCache.set(cache_key, cf)
      return
    }
  }
  err = &ConfigErr{When: time.Now(), What: "no match by source path."}
  return
}
