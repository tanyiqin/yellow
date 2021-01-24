package conf

import (
	"gopkg.in/yaml.v3"
	"os"
	"sync"
)

var (
	mutex sync.Mutex
	conf *totalConf
)

type totalConf struct {
	GateConf `yaml:"yellow"`
	EtcdConf `yaml:"etcd"`
}

type GateConf struct {
	InnerAddr string `yaml:"inneraddr"`
	OuterAddr string `yaml:"outeraddr"`
	GateID int `yaml:"gateid"`
}

type EtcdConf struct {
	endpoints []string `yaml:"endpoints"`
}

func Reload() error {
	mutex.Lock()
	defer mutex.Unlock()
	conf = nil
	return initConf()
}

func GetConf() *totalConf {
	mutex.Lock()
	defer mutex.Unlock()
	if conf == nil {
		_ = initConf()
	}
	return conf
}

func initConf() error {
	yFile, err := os.Open("conf/conf.yaml")
	defer yFile.Close()
	if err != nil {
		return err
	}
	yDecode := yaml.NewDecoder(yFile)
	return yDecode.Decode(&conf)
}