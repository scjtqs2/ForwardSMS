package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/smtp"

	log "github.com/sirupsen/logrus"
)

func sendNotification(config map[string]interface{}, sender, time, text, rule string) {
	message := fmt.Sprintf("触发规则: %s\n发送时间: %s\n发送人: %s\n短信内容: %s", rule, time, sender, text)

	notifyType, ok := config["notify"].(string)
	if !ok {
		log.Error("通知类型配置错误")
		return
	}

	switch notifyType {
	case "wechat":
		url, ok := config["url"].(string)
		if ok {
			sendWechat(url, sender, time, text, rule)
		}
	case "bark":
		url, ok := config["url"].(string)
		if ok {
			sendBark(url, "短信通知", message)
		}
	case "gotify":
		url, ok1 := config["url"].(string)
		token, ok2 := config["token"].(string)
		if ok1 && ok2 {
			sendGotify(url, token, "短信通知", message)
		}
	case "email":
		smtpHost, ok1 := config["smtp_host"].(string)
		smtpPort, ok2 := config["smtp_port"].(string)
		username, ok3 := config["username"].(string)
		password, ok4 := config["password"].(string)
		from, ok5 := config["from"].(string)
		to, ok6 := config["to"].(string)
		if ok1 && ok2 && ok3 && ok4 && ok5 && ok6 {
			sendEmail(smtpHost, smtpPort, username, password, from, to, "短信通知", message)
		}
	default:
		log.Warnf("未知的通知类型: %s", notifyType)
	}
}

func sendWechat(url string, SenderNumber string, ReceivingDateTime string, TextDecoded string, rule string) {
	smssend := fmt.Sprintf(`{"msgtype":"text","text":{"content":"触发规则: %s \n发送时间: %s \n发送人：%s \n短信内容：%s"}}`, rule, ReceivingDateTime, SenderNumber, TextDecoded)
	var jsonStr = []byte(smssend)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Errorf("创建微信请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("发送微信通知失败: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Infof("微信通知响应状态: %s", resp.Status)
}

func sendBark(url, title, body string) {
	payload := fmt.Sprintf(`{"title":"%s","body":"%s"}`, title, body)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		log.Errorf("创建Bark请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("发送Bark通知失败: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Info("Bark通知发送成功")
}

func sendGotify(url, token, title, message string) {
	payload := fmt.Sprintf(`{"title":"%s","message":"%s","priority":5}`, title, message)
	req, err := http.NewRequest("POST", url+"/message?token="+token, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		log.Errorf("创建Gotify请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("发送Gotify通知失败: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Info("Gotify通知发送成功")
}

func sendEmail(smtpHost, smtpPort, username, password, from, to, subject, body string) {
	auth := smtp.PlainAuth("", username, password, smtpHost)
	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		log.Errorf("邮件发送失败: %v", err)
		return
	}
	log.Info("邮件发送成功")
}
