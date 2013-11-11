package config

import (
  "testing"
)

//配置文件样例
const CONFIG_FILE = "../../etc/config_test.json"

var configFile ConfigFile

const CONFIG_COUNT = 6

//测试配置文件解析
func TestLoadConfig(t *testing.T) {
  if configFile = LoadConfigFile(CONFIG_FILE); configFile.Len() != CONFIG_COUNT {
    t.Errorf("解析配置错误")
  }
}

func BenchmarkLoadConfig(b *testing.B) {
  for i := 0; i < b.N; i++ {
    if configFile = LoadConfigFile(CONFIG_FILE); configFile.Len() != CONFIG_COUNT {
      b.Error("解析配置错误")
    }
  }
}

//测试根据源请求路径进行查找
func TestFindBySourcePath(t *testing.T) {
  config, _ := configFile.FindBySourcePath("/ticket/req.do")
  if config.Id != 90 {
    t.Errorf("根据源请求路径进行查找错误")
  }
}

func BenchmarkFindBySourcePath(b *testing.B) {
  for i := 0; i < b.N; i++ {
    config, _ := configFile.FindBySourcePath("/ticket/req.do")
    if config.Id != 90 {
      b.Errorf("根据源请求路径进行查找错误")
    }
  }
}


func TestFindBySourcePath2(t *testing.T) {
  if config, err := configFile.FindBySourcePath("/BB");err != nil{
    t.Errorf("根据源请求路径进行查找错误.A")
  }else if config.Id != 5000 {
    t.Errorf("根据源请求路径进行查找错误.B")
  }
}

func BenchmarkFindBySourcePath2(b *testing.B) {
  for i := 0; i < b.N; i++ {
    if config, err := configFile.FindBySourcePath("/BB"); err !=nil {
      b.Errorf("根据源请求路径进行查找错误.A")
    }else if config.Id != 5000 {
      b.Errorf("根据源请求路径进行查找错误.B")
    }
  }
}



//测试根据源参数进行查找
func TestFindBySourceParams(t *testing.T) {
  config, _ := configFile.FindBySourceParams([]string{"processcode=11002"})
  if config.Id != 2000 {
    t.Errorf("根据源参数进行查找错误")
  }
}

func BenchmarkFindBySourceParams(b *testing.B) {
  for i := 0; i < b.N; i++ {
    config, _ := configFile.FindBySourceParams([]string{"processcode=11002"})
    if config.Id != 2000 {
      b.Errorf("根据源参数进行查找错误")
    }
  }
}

func TestFindBySourceParams2(t *testing.T) {
  if config, err := configFile.FindBySourceParams([]string{"processcode=r1002","processcode=r2002"}); err != nil {
    t.Errorf("根据源参数进行查找错误.A")
  } else if config.Id != 2001 {
    t.Errorf("根据源参数进行查找错误.B")
  }

}

func BenchmarkFindBySourceParams2(b *testing.B) {
  for i := 0; i < b.N; i++ {
    if config, err := configFile.FindBySourceParams([]string{"processcode=r1002","processcode=r2002"}); err != nil {
      b.Errorf("根据源参数进行查找错误.A")
    } else if config != nil && config.Id != 2001 {
      b.Errorf("根据源参数进行查找错误.B")
    }
  }
}

//应该返回默认配置
func TestFindByParamsOrSourcePath_GotDefault(t *testing.T) {
  config := configFile.FindByParamsOrSourcePath([]string{"processcode=noexits"}, "/noexits")
  if config.TargetServer != "http://10.150.150.184" {
    t.Errorf("根据源参数或路径进行查找错误")
  }
}

//根据路径和参数进行匹配
func TestFindBySourcePathAndParams_Got_100(t *testing.T) {
  config, _ := configFile.FindBySourcePathAndParams([]string{"processcode=99999", "processcode=88888"}, "/ticket/req.do")
  if config.Id != 100 {
    t.Errorf("根据源参数和路径进行查找错误")
  }
}

func TestFindBySourcePathAndParams_Got_nil(t *testing.T) {
  config, _ := configFile.FindBySourcePathAndParams([]string{"processcode=no", "processcode=noconfig"}, "/ticket/req.do")
  if config != nil {
    t.Errorf("根据源参数和路径进行查找错误")
  }
}
