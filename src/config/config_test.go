package config

import (
	"testing"
)

//配置文件样例
const CONFIG_FILE = "../../etc/config_test.json"

var configFile ConfigFileT

//测试配置文件解析
func TestLoadConfig(t *testing.T) {
	if configFile = LoadConfigFile(CONFIG_FILE); configFile.Len() != 3 {
		t.Errorf("解析配置错误")
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if configFile = LoadConfigFile(CONFIG_FILE); configFile.Len() != 3 {
			b.Error("解析配置错误")
		}
	}
}

//测试根据源请求路径进行查找
func TestFindBySourcePath(t *testing.T) {
	config, _ := configFile.FindBySourcePath("/ticket/req.do")
	if config.Id != 100 {
		t.Errorf("根据源请求路径进行查找错误")
	}
}

func BenchmarkFindBySourcePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		config, _ := configFile.FindBySourcePath("/ticket/req.do")
		if config.Id != 100 {
			b.Errorf("根据源请求路径进行查找错误")
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
