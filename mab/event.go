package mab

import (
	"fmt"
	"github.com/importcjj/sensitive"
	"log"
)

func (rest *RestApi) checkConfig() {
	if rest.Cfg.ErrorTrace == nil {
		rest.Cfg.ErrorTrace = func(err error, event, from, router string) {
			log.Printf("[ab][%s] error:%s event:%s from:%s ", router, err, event, from)
		}
	}
	// 如果struct内联分隔符使用了.和_则抛出异常
	if rest.Cfg.StructDelimiter == "." || rest.Cfg.StructDelimiter == "_" {
		panic("StructDelimiter请勿使用.和_,因为.会转义 _是默认snake规则 使用会导致赋值异常 建议不设置默认为__")
	}
}

// 初始化敏感词库
// *.txt 每个关键词一行
func (rest *RestApi) initSensitive() {
	rest.sensitiveInstance = sensitive.New()
	if len(rest.Cfg.SensitiveUri) > 0 {
		for _, uri := range rest.Cfg.SensitiveUri {
			err := rest.sensitiveInstance.LoadNetWordDict(uri)
			if err != nil {
				panic(fmt.Errorf("获取敏感词库失败 连接:%s 错误:%v", uri, err))
			}
		}
	}
	if len(rest.Cfg.SensitiveWords) > 0 {
		rest.sensitiveInstance.AddWord(rest.Cfg.SensitiveWords...)
	}
}

func (rest *RestApi) runWordValid(words ...string) (bool, string) {
	for _, word := range words {
		pass, firstWord := rest.sensitiveInstance.Validate(word)
		if pass != true {
			return false, firstWord
		}
	}
	return true, ""
}
