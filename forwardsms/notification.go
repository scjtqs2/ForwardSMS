package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func sendNotification(config map[string]interface{}, sender, time, text, rule string, smsReq SMSRequest) {
	message := fmt.Sprintf("触发规则: %s\n发送时间: %s\n发送人: %s \nphoneID: %s\n短信内容: %s\nSource: %s", rule, time, sender, smsReq.PhoneID, text, smsReq.Source)
	messagePhone := fmt.Sprintf("%s\n%s\n%s\n%s", text, smsReq.PhoneID, smsReq.Time, smsReq.Source)

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
		// messagePhone = fmt.Sprintf("%s%s%s%s", text, smsReq.PhoneID, smsReq.Time, smsReq.Source)
		url, ok := config["url"].(string)
		if ok {
			sendBark(url, sender, messagePhone)
		}
	case "gotify":
		url, ok1 := config["url"].(string)
		token, ok2 := config["token"].(string)
		if ok1 && ok2 {
			sendGotify(url, token, sender, messagePhone)
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
	case "qq":
		qq, ok1 := config["qq"].(string)
		token, ok2 := config["token"].(string)
		if ok1 && ok2 {
			sendQQPush(fmt.Sprintf("短信通知\n%%", message), qq, token)
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

// BarkRequest Bark请求参数
type BarkRequest struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	IsArchive int    `json:"isArchive,omitempty"`
	Group     string `json:"group,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Level     string `json:"level,omitempty"`
	Sound     string `json:"sound,omitempty"`
	Badge     string `json:"badge,omitempty"`
	URL       string `json:"url,omitempty"`
	Call      string `json:"call,omitempty"`
	Copy      string `json:"copy,omitempty"`
	AutoCopy  int    `json:"autoCopy,omitempty"`
}

func sendBark(url, title, body string) {
	// 构建请求参数
	msgMap := BarkRequest{
		Title:     title,
		Body:      body,
		IsArchive: 1,
	}
	// 检测验证码模式
	pattern := `(?i)(回复)?(验证码|授权码|校验码|检验码|确认码|激活码|动态码|安全码|(验证)?代码|校验代码|检验代码|激活代码|确认代码|动态代码|安全代码|登入码|认证码|识别码|短信口令|动态密码|交易码|上网密码|动态口令|随机码|驗證碼|授權碼|校驗碼|檢驗碼|確認碼|激活碼|動態碼|(驗證)?代碼|校驗代碼|檢驗代碼|確認代碼|激活代碼|動態代碼|登入碼|認證碼|識別碼|一次性密码|CODE|Verification)`
	matched, _ := regexp.MatchString(pattern, body)
	if matched {
		// 提取验证码
		code := extractVerificationCode(body)
		if code != "" {
			msgMap.Copy = code
			msgMap.AutoCopy = 1
		}
	}
	// 序列化请求数据
	requestMsg, err := json.Marshal(msgMap)
	if err != nil {
		log.Errorf("序列化请求数据失败: %v", err)
		return
	}

	log.Infof("Bark请求数据: %s", string(requestMsg))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestMsg))
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

type GotifyRequest struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority int    `json:"priority,omitempty"`
}

func sendGotify(url, token, title, message string) {
	fmt.Println(url, token, title, message)
	msg := GotifyRequest{
		Title:    title,
		Message:  message,
		Priority: 9,
	}
	payloadBytes, _ := json.Marshal(msg)
	req, err := http.NewRequest("POST", url+"/message?token="+token, bytes.NewBuffer(payloadBytes))
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

// extractVerificationCode 从内容中提取验证码
func extractVerificationCode(content string) string {
	// 第一种模式匹配
	pattern1 := `(.*)((代|授权|验证|动态|校验)码|[【\[].[】\]]|[Cc][Oo][Dd][Ee]|[Vv]erification\s?([Cc]ode)?)\s?(G-|<#>)?([:：\s是为]|[Ii][Ss]){0,3}[\(（\[【{「]?(([0-9\s]{4,7})|([\dA-Za-z]{5,6})(?!([Vv]erification)?([Cc][Oo][Dd][Ee])|:))[」}】\]]?[）\)]?(?=([^0-9a-zA-Z]|$))(.*)`
	re1 := regexp.MustCompile(pattern1)
	matches1 := re1.FindStringSubmatch(content)
	if len(matches1) > 7 {
		code := strings.TrimSpace(matches1[7])
		if code != "" {
			return code
		}
	}

	// 第二种模式匹配
	pattern2 := `\D*[\(（\[【{「]?([0-9]{3}\s?[0-9]{1,3})[」}】\]]?[）\)]?(?=.*((代|授权|验证|动态|校验)码|[【\[].[】\]]|[Cc][Oo][Dd][Ee]|[Vv]erification\s?([Cc]ode)?))(.*)`
	re2 := regexp.MustCompile(pattern2)
	matches2 := re2.FindStringSubmatch(content)
	if len(matches2) > 1 {
		code := strings.TrimSpace(matches2[1])
		if code != "" {
			return code
		}
	}

	return ""
}

type PostData map[string]interface{}

func sendQQPush(token, cqq, msg string) {
	log.Infof("发送QQPush通知: token=%s, cqq=%s, msg=%s", token, cqq, msg)

	posturl := fmt.Sprintf("https://wx.scjtqs.com/qq/push/pushMsg?token=%s", token)
	header := make(http.Header)
	// header.Set("Users-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:32.0) Gecko/20100101 Firefox/32.0")
	header.Set("content-type", "application/json")

	postdata, err := json.Marshal(PostData{
		"qq": cqq,
		"content": []PostData{
			{
				"msgtype": "text",
				"text":    msg,
			},
		},
		"token": token,
	})
	if err != nil {
		log.Errorf("序列化QQPush请求数据失败: %v", err)
		return
	}

	req, err := http.NewRequest("POST", posturl, bytes.NewBuffer(postdata))
	if err != nil {
		log.Errorf("创建QQPush请求失败: %v", err)
		return
	}
	req.Header = header

	client := &http.Client{Timeout: time.Second * 6}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("发送QQPush通知失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("读取QQPush响应失败: %v", err)
		return
	}

	log.Infof("QQPush通知发送成功, 响应: %s", string(body))
}
