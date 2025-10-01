#!/bin/bash

# 短信转发脚本 - 专用于 Go HTTP 转发服务
# 环境变量配置:
# FORWARD_URL: 转发服务URL (默认: http://forwardsms:8080)
# FORWARD_SECRET: 可选认证密钥

LOG_FILE="/data/log/forward.log"
DATABASE="/data/db/sms.db"
LOCK_FILE="/tmp/sms_forward.lock"

# 创建日志目录
mkdir -p "$(dirname "$LOG_FILE")"

# 记录日志函数
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG_FILE"
}

# 文件锁函数，防止并发执行
acquire_lock() {
    local timeout=30
    local start_time=$(date +%s)

    while [ -f "$LOCK_FILE" ]; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))

        if [ $elapsed -gt $timeout ]; then
            log "❌ 获取锁超时，跳过处理"
            return 1
        fi
        sleep 1
    done

    touch "$LOCK_FILE"
    return 0
}

release_lock() {
    rm -f "$LOCK_FILE"
}

# 从环境变量读取配置
FORWARD_URL="${FORWARD_URL:-http://forwardsms:8080/api/v1/sms/receive}"
FORWARD_SECRET="${FORWARD_SECRET:-}"
FORWARD_TIMEOUT="${FORWARD_TIMEOUT:-30}"
PHONE_ID="${PHONE_ID:-default-phone}"

# 状态文件，记录最后处理的短信ID
STATE_FILE="/data/db/last_processed_id"
LAST_PROCESSED_ID=0

# 读取最后处理的ID
read_last_processed_id() {
    if [ -f "$STATE_FILE" ]; then
        LAST_PROCESSED_ID=$(cat "$STATE_FILE" 2>/dev/null || echo 0)
    fi
    echo $LAST_PROCESSED_ID
}

# 保存最后处理的ID
save_last_processed_id() {
    local id="$1"
    echo "$id" > "$STATE_FILE"
}

# 获取未处理的短信
get_unprocessed_sms() {
    local last_id=$(read_last_processed_id)
    sqlite3 "$DATABASE" "SELECT ID, TextDecoded, SenderNumber, ReceivingDateTime FROM inbox WHERE ID > $last_id ORDER BY ID ASC;"
}

# 构建 JSON 数据并转发到 Go 服务
forward_to_golang_service() {
    local id="$1"
    local text="$2"
    local number="$3"
    local time="$4"

    # 构建 JSON 数据
    local json_data
    json_data=$(cat <<EOF
{
    "secret": "${FORWARD_SECRET}",
    "number": "${number}",
    "time": "${time}",
    "text": "${text}",
    "source": "gammu-smsd",
    "phone_id": "${PHONE_ID}",
    "sms_id": ${id},
    "timestamp": "$(date -Iseconds)"
}
EOF
    )

    log "尝试转发短信ID: $id 到: $FORWARD_URL"
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
        log "✓ 短信ID: $id 成功转发到 Go 服务 (HTTP $http_code)"
        if [ -n "$response_body" ]; then
            log "  服务响应: $response_body"
        fi
        return 0
    else
        log "✗ 短信ID: $id 转发失败 - HTTP 状态码: $http_code"
        if [ -n "$response_body" ]; then
            log "  错误响应: $response_body"
        fi
        return 1
    fi
}

# 处理单条短信
process_single_sms() {
    local id="$1"
    local text="$2"
    local number="$3"
    local time="$4"

    # 转义 JSON 特殊字符
    local escaped_text=$(echo "$text" | sed 's/"/\\"/g' | sed 's/\\/\\\\/g')

    log "处理短信ID: $id - 发件人: $number, 时间: $time"

    # 转发到 Go 服务
    if forward_to_golang_service "$id" "$escaped_text" "$number" "$time"; then
        # 只有转发成功才更新状态
        save_last_processed_id "$id"
        log "✅ 短信ID: $id 处理完成"
        return 0
    else
        log "❌ 短信ID: $id 转发失败，保留状态等待重试"
        return 1
    fi
}

# 主函数
main() {
    log "📱 开始检查未处理短信..."

    # 获取文件锁，防止并发执行
    if ! acquire_lock; then
        log "⚠️ 已有处理进程在运行，退出"
        exit 0
    fi

    # 确保锁被释放
    trap release_lock EXIT

    # 获取未处理的短信
    local unprocessed_sms
    unprocessed_sms=$(get_unprocessed_sms)

    if [ -z "$unprocessed_sms" ]; then
        log "ℹ️ 没有未处理的短信"
        release_lock
        exit 0
    fi

    # 统计处理数量
    local processed_count=0
    local failed_count=0

    # 处理每一条短信
    echo "$unprocessed_sms" | while IFS='|' read -r id text number time; do
        if [ -n "$id" ] && [ -n "$text" ] && [ -n "$number" ] && [ -n "$time" ]; then
            if process_single_sms "$id" "$text" "$number" "$time"; then
                processed_count=$((processed_count + 1))
            else
                failed_count=$((failed_count + 1))
                # 如果一条失败，停止处理后续短信（避免ID跳跃）
                log "⚠️ 短信ID: $id 处理失败，停止处理后续短信"
                break
            fi
        fi
    done

    log "📊 处理完成 - 成功: $processed_count, 失败: $failed_count"
}

# 执行主函数
main "$@"