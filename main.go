package yellow

import (
	"yellow/conf"
	"yellow/log"

)

func main() {
	// 解析配置
	config := conf.GetConf()
	if config == nil {
		log.Panic("err in parse config")
	}
}
