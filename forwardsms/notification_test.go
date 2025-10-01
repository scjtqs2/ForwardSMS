package main

import (
	"testing"

	"github.com/spf13/viper"
)

/**
 * @author scjtqs
 * @email scjtqs@qq.com
 */

func TestInitConfig(t *testing.T) {
	viperconfig = viper.New()
	viperconfig.SetConfigName("test")
	viperconfig.SetConfigType("yaml")
	viperconfig.AddConfigPath("../data/config")
	if err := viperconfig.ReadInConfig(); err != nil {
		t.Fatalf("读取推送配置失败: %v", err)
	}
	if err := viperconfig.Unmarshal(&config); err != nil {
		t.Fatalf("解析推送配置失败: %v", err)
	}
	t.Log("读取推送配置完成")
}

func TestSendNotification(t *testing.T) {
	TestInitConfig(t)
	msg := SMSRequest{
		Number:  "18611111111",
		Time:    "2025-10-01T08:49:44Z",
		Text:    "你好啊!!!",
		Source:  "forward test",
		PhoneID: "SMS1_123456789",
	}
	err := processSMS(msg.Number, msg.Time, msg.Text, msg)
	if err != nil {
		t.Fatalf("推送错误 %s", err.Error())
	}
}
