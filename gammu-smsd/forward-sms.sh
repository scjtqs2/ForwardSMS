#!/bin/bash

# 短信转发脚本 - 专用于 Go HTTP 转发服务
# 环境变量配置:
# FORWARD_URL: 转发服务URL (默认: http://forwardsms:8080)
# FORWARD_SECRET: 可选认证密钥

LOG_FILE="/data/log/forward.log"
DATABASE="/data/db/sms.db"

# 创建日志目录
mkdir -p "$(dirname "$LOG_FILE")"

# 记录日志函数
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG_FILE"
}

# 从环境变量读取配置
FORWARD_URL="${FORWARD_URL:-http://forwardsms:8080/api/v1/sms/receive}"
FORWARD_SECRET="${FORWARD_SECRET:-}"
FORWARD_TIMEOUT="${FORWARD_TIMEOUT:-30}"

# 从数据库获取最新短信
get_latest_sms() {
    sqlite3 "$DATABASE" "SELECT Text, Number, InsertIntoDatetime FROM inbox ORDER BY InsertIntoDatetime DESC LIMIT 1;"
}

# 构建 JSON 数据并转发到 Go 服务
forward_to_golang_service() {
    local text="$1"
    local number="$2"
    local time="$3"

    # 构建 JSON 数据
    local json_data
    json_data=$(cat <<EOF
{
    "secret": "${FORWARD_SECRET}",
    "number": "${number}",
    "time": "${time}",
    "text": "${text}",
    "source": "gammu-smsd",
    "timestamp": "$(date -Iseconds)"
}
EOF
    )

    log "尝试转发到: $FORWARD_URL"
    log "短信数据 - 发件人: $number, 时间: $time, 内容长度: ${#text} 字符"

    # 使用 curl 发送 POST 请求
    local response
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -H "User-Agent: Gammu-SMSD/1.0" \
        -d "$json_data" \
        --connect-timeout 10 \
        --max-time "$FORWARD_TIMEOUT" \
        "$FORWARD_URL")

    local http_code
    http_code=$(echo "$response" | tail -n1)
    local response_body
    response_body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 200 ]; then
        log "✓ 短信成功转发到 Go 服务 (HTTP $http_code)"
        if [ -n "$response_body" ]; then
            log "  服务响应: $response_body"
        fi
        return 0
    else
        log "✗ 转发失败 - HTTP 状态码: $http_code"
        if [ -n "$response_body" ]; then
            log "  错误响应: $response_body"
        fi
        return 1
    fi
}

# 主函数
main() {
    log "📱 收到新短信，开始处理..."

    # 获取最新短信
    SMS_DATA=$(get_latest_sms)
    if [ -z "$SMS_DATA" ]; then
        log "❌ 错误：无法从数据库获取短信数据"
        exit 1
    fi

    # 解析短信数据 (SQLite 输出格式: text|number|datetime)
    SMS_TEXT=$(echo "$SMS_DATA" | cut -d'|' -f1)
    SMS_NUMBER=$(echo "$SMS_DATA" | cut -d'|' -f2)
    SMS_TIME=$(echo "$SMS_DATA" | cut -d'|' -f3)

    # 转义 JSON 特殊字符
    SMS_TEXT_ESCAPED=$(echo "$SMS_TEXT" | sed 's/"/\\"/g' | sed 's/\\/\\\\/g')

    log "处理短信 - 发件人: $SMS_NUMBER, 时间: $SMS_TIME"

    # 转发到 Go 服务
    if forward_to_golang_service "$SMS_TEXT_ESCAPED" "$SMS_NUMBER" "$SMS_TIME"; then
        log "✅ 短信处理流程完成"
    else
        log "❌ 短信转发失败，将重试下次"
        exit 1
    fi
}

# 执行主函数
main "$@"