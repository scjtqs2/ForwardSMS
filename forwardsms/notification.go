package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
)

func sendNotification(config map[string]interface{}, sender, time, text, rule string) {
	message := fmt.Sprintf("触发规则: %s\n发送时间: %s\n发送人: %s\n短信内容: %s", rule, time, sender, text)

	switch config["notify"].(string) {
	case "wechat":
		sendWechat(config["url"].(string), sender, time, text, rule)
	case "bark":
		sendBark(config["url"].(string), "短信通知", message)
	case "gotify":
		sendGotify(config["url"].(string), config["token"].(string), "短信通知", message)
	case "email":
		sendEmail(
			config["smtp_host"].(string),
			config["smtp_port"].(string),
			config["username"].(string),
			config["password"].(string),
			config["from"].(string),
			config["to"].(string),
			"短信通知",
			message,
		)
	}
}

func sendWechat(url string, SenderNumber string, ReceivingDateTime string, TextDecoded string, rule string) {
	smssend := fmt.Sprintf(`{"msgtype":"text","text":{"content":"触发规则: %s \n发送时间: %s \n发送人：%s \n短信内容：%s" \n}}`, rule, ReceivingDateTime, SenderNumber, TextDecoded)
	var jsonStr = []byte(smssend)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

// Bark 推送
func sendBark(url, title, body string) {
	payload := fmt.Sprintf(`{"title":"%s","body":"%s"}`, title, body)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Bark 发送失败:", err)
		return
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Bark response:", string(data))
}

// Gotify 推送
func sendGotify(url, token, title, message string) {
	payload := fmt.Sprintf(`{"title":"%s","message":"%s","priority":5}`, title, message)
	req, _ := http.NewRequest("POST", url+"/message?token="+token, bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Gotify 发送失败:", err)
		return
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Gotify response:", string(data))
}

// 邮件通知
func sendEmail(smtpHost, smtpPort, username, password, from, to, subject, body string) {
	auth := smtp.PlainAuth("", username, password, smtpHost)
	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		fmt.Println("邮件发送失败:", err)
		return
	}
	fmt.Println("邮件发送成功")
}
