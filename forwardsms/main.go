package main

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	viperconfig *viper.Viper
	config      = map[string]interface{}{}
	router      *gin.Engine
	lastSMSID   int
)

// SMSRequest 接收来自 gammu-smsd 的请求结构
type SMSRequest struct {
	Secret    string `json:"secret"`
	Number    string `json:"number"`
	Time      string `json:"time"`
	Text      string `json:"text"`
	Source    string `json:"source"`
	PhoneID   string `json:"phone_id"`
	SMSID     int    `json:"sms_id"`
	Timestamp string `json:"timestamp"`
}

// SMSResponse 响应结构
type SMSResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	ID      int    `json:"id,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Time    string `json:"time"`
	Version string `json:"version"`
}

// Config 服务器配置
type Config struct {
	Server struct {
		Port   string `yaml:"port"`
		Secret string `yaml:"secret"`
	} `yaml:"server"`
}

func main() {
	// 初始化日志
	log.SetFormatter(&log.JSONFormatter{})
	log.Info("启动短信转发服务...")

	// 读取配置文件
	if err := initConfig(); err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	// 初始化 Gin
	initGin()

	// 启动 HTTP 服务器
	startHTTPServer()
}

func initConfig() error {
	// 读取发送配置文件
	viperconfig = viper.New()
	viperconfig.SetConfigName("forward")
	viperconfig.SetConfigType("yaml")
	viperconfig.AddConfigPath("/data/config")
	if err := viperconfig.ReadInConfig(); err != nil {
		return fmt.Errorf("读取推送配置失败: %v", err)
	}
	if err := viperconfig.Unmarshal(&config); err != nil {
		return fmt.Errorf("解析推送配置失败: %v", err)
	}
	log.Info("读取推送配置完成")
	return nil
}

func initGin() {
	// 设置 Gin 模式
	if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "true" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	router = gin.Default()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	// 设置路由
	setupRoutes()
}

func setupRoutes() {
	// API v1 分组
	v1 := router.Group("/api/v1")
	{
		// 短信接收端点
		v1.POST("/sms", smsHandler)
		v1.POST("/sms/receive", smsHandler) // 兼容性端点

		// 管理端点
		v1.GET("/health", healthHandler)
		v1.GET("/status", statusHandler)
		v1.POST("/test", testHandler)
	}

	// 根路径重定向到健康检查
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/api/v1/health")
	})
}

// CORSMiddleware 跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func startHTTPServer() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Infof("HTTP 服务启动，监听端口 %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("HTTP 服务启动失败: %v", err)
	}
}

// healthHandler 健康检查端点
func healthHandler(c *gin.Context) {
	response := HealthResponse{
		Status:  "healthy",
		Service: "sms-forward",
		Time:    time.Now().Format(time.RFC3339),
		Version: "1.0.0",
	}
	c.JSON(http.StatusOK, response)
}

// statusHandler 服务状态端点
func statusHandler(c *gin.Context) {
	configCount := len(config)

	c.JSON(http.StatusOK, gin.H{
		"status":            "running",
		"last_processed_id": lastSMSID,
		"rule_count":        configCount,
		"timestamp":         time.Now().Format(time.RFC3339),
	})
}

// testHandler 测试端点
func testHandler(c *gin.Context) {
	var testReq struct {
		Number string `json:"number"`
		Text   string `json:"text"`
	}

	if err := c.ShouldBindJSON(&testReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "无效的请求数据",
		})
		return
	}

	// 使用测试数据处理短信
	if err := processSMS(testReq.Number, time.Now().Format("2006-01-02 15:04:05"), testReq.Text); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "测试短信处理完成",
	})
}

// smsHandler 处理来自 gammu-smsd 的短信推送
func smsHandler(c *gin.Context) {
	var smsReq SMSRequest
	if err := c.ShouldBindJSON(&smsReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "无效的 JSON 数据: " + err.Error(),
		})
		return
	}

	// 验证密钥（可选）
	if err := validateSecret(smsReq.Secret); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "认证失败",
		})
		return
	}
	lastSMSID = smsReq.SMSID
	log.WithFields(log.Fields{
		"number":   smsReq.Number,
		"time":     smsReq.Time,
		"source":   smsReq.Source,
		"sms_id":   smsReq.SMSID,
		"phone_id": smsReq.PhoneID,
	}).Info("收到短信推送")

	// 处理短信转发
	if err := processSMS(smsReq.Number, smsReq.Time, smsReq.Text); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "处理短信失败: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "短信接收并处理成功",
	})
}

func validateSecret(secret string) error {
	expectedSecret := os.Getenv("FORWARD_SECRET")
	if expectedSecret != "" && secret != expectedSecret {
		return fmt.Errorf("密钥验证失败")
	}
	return nil
}

func processSMS(sender, time, text string) error {
	log.WithFields(log.Fields{
		"sender": sender,
		"time":   time,
		"text":   text,
	}).Info("开始处理短信")

	// 遍历所有配置的转发规则
	for name, cfg := range config {
		c, ok := cfg.(map[string]interface{})
		if !ok {
			log.Warnf("配置格式错误: %s", name)
			continue
		}

		rule, ok := c["rule"].(string)
		if !ok {
			log.Warnf("规则配置错误: %s", name)
			continue
		}

		ruleType, ok := c["type"].(string)
		if !ok {
			log.Warnf("类型配置错误: %s", name)
			continue
		}

		// 根据规则类型匹配
		if shouldSendNotification(ruleType, rule, text) {
			log.Infof("触发规则: %s, 类型: %s", name, ruleType)
			sendNotification(c, sender, time, text, rule)
		}
	}

	return nil
}

func shouldSendNotification(ruleType, rule, text string) bool {
	switch ruleType {
	case "all":
		return true
	case "keyword":
		return strings.Contains(text, rule)
	case "regex":
		matched, err := regexp.MatchString(rule, text)
		if err != nil {
			log.Errorf("正则表达式错误: %v", err)
			return false
		}
		return matched
	default:
		log.Warnf("未知的规则类型: %s", ruleType)
		return false
	}
}
