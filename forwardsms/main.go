package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"regexp"
	"strings"
)

var (
	viperStatus *viper.Viper
	viperconfig *viper.Viper
	config      = map[string]interface{}{}
	status      = map[string]interface{}{}
	localCron   *cron.Cron
)

func main() {
	// 读取发送配置文件
	viperconfig = viper.New()
	viperconfig.SetConfigName("forward")
	viperconfig.SetConfigType("yaml")
	viperconfig.AddConfigPath("/data/config")
	err := viperconfig.ReadInConfig()
	if err != nil {
		log.Fatalf("read config failed: %v", err)
		return
	}
	err = viperconfig.Unmarshal(&config)
	if err != nil {
		log.Fatalf("read config failed: %v", err)
		return
	}
	log.Info("读取推送配置完成")
	// 读取当前发送规则配置文件
	viperStatus = viper.New()
	viperStatus.SetConfigName("status")
	viperStatus.SetConfigType("yaml")
	viperStatus.AddConfigPath("/data/config")
	err = viperStatus.ReadInConfig()
	if err != nil {
		log.Fatalf("Fatal error config file: %s \n", err)
		return
	}
	err = viperStatus.Unmarshal(&status)
	if err != nil {
		log.Fatalf("Fatal error config file: %s \n", err)
		return
	}
	log.Info("读取status配置完成")
	// 写入测试
	// fmt.Println(viperStatus.Get("id"))
	// viperStatus.Set("id", 2)
	// if err := viperStatus.WriteConfig(); err != nil {
	// 	fmt.Println(err)
	// }
	// 定时任务开启
	localCron = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	_, err = localCron.AddFunc("*/5 * * * * *", readSms)
	if err != nil {
		log.Fatalf("Fatal error add corn: %s \n", err)
		return
	}
	log.Info("定时任务开启完成，每5秒查询一次")
	localCron.Run()
}

func readSms() {
	log.Info("readSms start")
	defer log.Info("readSms end")

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
