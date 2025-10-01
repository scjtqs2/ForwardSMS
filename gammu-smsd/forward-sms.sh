#!/bin/bash

# 短信转发脚本 - 专用于文件方式的 Gammu 配置
# 环境变量配置:
# FORWARD_URL: 转发服务URL (默认: http://forwardsms:8080)
# FORWARD_SECRET: 可选认证密钥

LOG_FILE="/data/log/forward.log"
INBOX_DIR="/data/sms/inbox"
PROCESSED_DIR="/data/sms/processed"
LOCK_FILE="/tmp/sms_forward.lock"

# 创建必要的目录
mkdir -p "$(dirname "$LOG_FILE")"
mkdir -p "$PROCESSED_DIR"
mkdir -p "$INBOX_DIR"

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

# 调试函数：查看文件实际内容
debug_file_content() {
    local file="$1"
    log "🔍 调试文件内容: $file"
    log "   文件大小: $(wc -c < "$file") 字节"
    log "   文件行数: $(wc -l < "$file") 行"
    log "   文件内容（原始）:"
    hexdump -C "$file" | head -10 >> "$LOG_FILE"
    log "   文件内容（文本）:"
    cat "$file" | sed 's/^/      /' >> "$LOG_FILE"
}

# 解析短信文件内容 - 改进版本
parse_sms_file() {
    local file="$1"

    # Gammu 文件格式通常包含这些字段
    local number=""
    local text=""
    local time=""
    local sms_id=""

    # 从文件名提取信息
    sms_id=$(basename "$file")

    # 从文件名解析时间 (格式: IN20251001_183701_00_+8618628287642_00.txt)
    if [[ "$sms_id" =~ IN([0-9]{8})_([0-9]{6}) ]]; then
        local date_part="${BASH_REMATCH[1]}"  # 20251001
        local time_part="${BASH_REMATCH[2]}"  # 183701
        time="${date_part:0:4}-${date_part:4:2}-${date_part:6:2} ${time_part:0:2}:${time_part:2:2}:${time_part:4:2}"
    fi

    # 从文件名解析号码 (格式: IN20251001_183701_00_+8618628287642_00.txt)
    if [[ "$sms_id" =~ _([+0-9]+)_ ]]; then
        number="${BASH_REMATCH[1]}"
    fi

    # 读取文件内容
    if [ -f "$file" ]; then
        # 调试：查看文件实际内容
        debug_file_content "$file"

        # 读取整个文件内容作为短信正文
        text=$(cat "$file" | tr -d '\r' | sed '/^$/d')

        # 如果从文件名中没解析出号码，尝试从文件内容第一行读取
        if [ -z "$number" ] || [ "$number" == "还差还差哈哈哈" ]; then
            # 可能是文件格式不同，尝试其他解析方式
            local first_line=$(head -1 "$file" | tr -d '\r\n')
            if [[ "$first_line" =~ ^[+0-9]+$ ]]; then
                number="$first_line"
                # 如果第一行是号码，那么短信内容从第二行开始
                text=$(tail -n +2 "$file" | tr -d '\r' | sed '/^$/d')
            else
                # 第一行不是号码，整个文件都是内容
                text="$first_line$(tail -n +2 "$file" | tr -d '\r' | sed '/^$/d')"
            fi
        fi

        # 如果没有明确的时间，使用文件修改时间
        if [ -z "$time" ]; then
            time=$(date -r "$file" "+%Y-%m-%d %H:%M:%S")
        fi

        echo "$sms_id|$number|$text|$time"
    else
        echo ""
    fi
}

# 构建 JSON 数据并转发到 Go 服务
forward_to_golang_service() {
    local sms_id="$1"
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
    "sms_id": "${sms_id}",
    "timestamp": "$(date -Iseconds)"
}
EOF
    )

    log "尝试转发短信: $sms_id 到: $FORWARD_URL"
    log "短信数据 - 发件人: $number, 时间: $time, 内容长度: ${#text} 字符"
    log "短信内容: $text"

    # 使用 curl 发送 POST 请求
    local response
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -H "User-Agent: Gammu-SMSD/1.0" \
        -d "$json_data" \
        --connect-timeout 10 \
        -H "X-Forward-Secret: ${FORWARD_SECRET}" \
        --max-time "$FORWARD_TIMEOUT" \
        "$FORWARD_URL")

    local http_code
    http_code=$(echo "$response" | tail -n1)
    local response_body
    response_body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 200 ]; then
        log "✓ 短信: $sms_id 成功转发到 Go 服务 (HTTP $http_code)"
        if [ -n "$response_body" ]; then
            log "  服务响应: $response_body"
        fi
        return 0
    else
        log "✗ 短信: $sms_id 转发失败 - HTTP 状态码: $http_code"
        if [ -n "$response_body" ]; then
            log "  错误响应: $response_body"
        fi
        return 1
    fi
}

# 处理单条短信文件
process_single_sms() {
    local file="$1"

    # 解析短信文件
    local parsed_data
    parsed_data=$(parse_sms_file "$file")

    if [ -z "$parsed_data" ]; then
        log "❌ 无法解析短信文件: $file"
        return 1
    fi

    # 提取解析的数据
    IFS='|' read -r sms_id number text time <<< "$parsed_data"

    log "处理短信: $sms_id - 发件人: $number, 时间: $time, 内容: $text"

    # 检查号码是否有效
    if [ -z "$number" ] || [[ ! "$number" =~ ^[+0-9] ]]; then
        log "⚠️ 号码格式无效: '$number'，尝试从内容中提取"
        # 这里可以添加从内容中提取号码的逻辑
    fi

    # 转义 JSON 特殊字符
    local escaped_text=$(echo "$text" | sed 's/"/\\"/g' | sed 's/\\/\\\\/g')

    # 转发到 Go 服务
    if forward_to_golang_service "$sms_id" "$escaped_text" "$number" "$time"; then
        # 只有转发成功才移动文件
        mv "$file" "$PROCESSED_DIR/"
        log "✅ 短信: $sms_id 处理完成，文件已移动到 processed 目录"
        return 0
    else
        log "❌ 短信: $sms_id 转发失败，保留文件等待重试"
        return 1
    fi
}

# 获取未处理的短信文件
get_unprocessed_sms_files() {
    # 查找 inbox 目录下所有文件，按修改时间排序
    find "$INBOX_DIR" -maxdepth 1 -type f -name "*.txt" | sort
}

# 主函数
main() {
    log "📱 开始检查未处理短信..."

    # 检查 inbox 目录是否存在
    if [ ! -d "$INBOX_DIR" ]; then
        log "❌ Inbox 目录不存在: $INBOX_DIR"
        exit 1
    fi

    # 获取文件锁，防止并发执行
    if ! acquire_lock; then
        log "⚠️ 已有处理进程在运行，退出"
        exit 0
    fi

    # 确保锁被释放
    trap release_lock EXIT

    # 获取未处理的短信文件
    local unprocessed_files
    unprocessed_files=$(get_unprocessed_sms_files)

    if [ -z "$unprocessed_files" ]; then
        log "ℹ️ 没有未处理的短信"
        release_lock
        exit 0
    fi

    # 统计处理数量
    local processed_count=0
    local failed_count=0

    # 处理每一个短信文件
    while IFS= read -r file; do
        if [ -f "$file" ]; then
            if process_single_sms "$file"; then
                processed_count=$((processed_count + 1))
            else
                failed_count=$((failed_count + 1))
                log "⚠️ 短信文件处理失败: $file，继续处理下一个"
            fi
        fi
    done <<< "$unprocessed_files"

    log "📊 处理完成 - 成功: $processed_count, 失败: $failed_count"
}

# 执行主函数
main "$@"