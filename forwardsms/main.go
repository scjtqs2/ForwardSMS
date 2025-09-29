package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 读取发送配置文件
	config := map[string]interface{}{}
	viperconfig := viper.New()
	viperconfig.SetConfigName("forward")
	viperconfig.SetConfigType("yaml")
	viperconfig.AddConfigPath("/data/config")
	viperconfig.ReadInConfig()
	viperconfig.Unmarshal(&config)

	// 读取当前发送规则配置文件
	status := map[string]interface{}{}
	viperStatus := viper.New()
	viperStatus.SetConfigName("status")
	viperStatus.SetConfigType("yaml")
	viperStatus.AddConfigPath("/data/config")
	viperStatus.ReadInConfig()
	viperStatus.Unmarshal(&status)

	// 写入测试
	// fmt.Println(viperStatus.Get("id"))
	// viperStatus.Set("id", 2)
	// if err := viperStatus.WriteConfig(); err != nil {
	// 	fmt.Println(err)
	// }

	// 数据库查询
	db, err := gorm.Open(sqlite.Open("/data/db/sms.db"), &gorm.Config{})
	if err != nil {
		panic("数据库文件不存在")
	}
	inbox := []map[string]interface{}{}

	db.Debug().Select("ID", "TextDecoded", "SenderNumber", "ReceivingDateTime").Table("inbox").Where("id > ?", viperStatus.Get("id")).Find(&inbox)

	// 循环获取到的所有短信内容
	for _, inboxfor := range inbox {
		sender := inboxfor["SenderNumber"].(string)
		time := inboxfor["ReceivingDateTime"].(string)
		text := inboxfor["TextDecoded"].(string)

		for name, cfg := range config {
			c := cfg.(map[string]interface{})
			rule := c["rule"].(string)

			switch c["type"].(string) {
			case "all":
				sendNotification(c, sender, time, text, rule)
				fmt.Println("触发 all 配置:", name)

			case "keyword":
				if strings.Contains(text, rule) {
					sendNotification(c, sender, time, text, rule)
					fmt.Println("触发 keyword 配置:", name)
				}

			case "regex":
				matched, err := regexp.MatchString(rule, text)
				if err != nil {
					fmt.Println("正则错误:", err)
					continue
				}
				if matched {
					sendNotification(c, sender, time, text, rule)
					fmt.Println("触发 regex 配置:", name)
				}
			}
		}

		// 写入最后一次发短信的ID
		viperStatus.Set("id", inboxfor["ID"])
		if err := viperStatus.WriteConfig(); err != nil {
			fmt.Println(err)
		}
	}
}
