#!/bin/bash

set -e

# 替换 USB 端口配置
if [ -n "$USB_PORT" ]; then
    echo "配置 USB 端口: $USB_PORT"
    sed -i "s|%USB_PORT%|$USB_PORT|g" /etc/gammu-smsd/gammu-smsdrc
else
    echo "警告: USB_PORT 环境变量未设置，使用默认配置"
fi

if [ -n "$PHONE_ID" ]; then
  echo "配置PHONE_ID：$PHONE_ID"
  sed -i "s|%PHONE_ID%|$PHONE_ID|g" /etc/gammu-smsd/gammu-smsdrc
else
  echo "警告: PHONE_ID 环境变量未设置，使用默认配置"
fi

if [ -n "$ATCONNECTION" ]; then
   echo "配置 AT 连接: $ATCONNECTION"
   sed -i "s|%ATCONNECTION%|$ATCONNECTION|g" /etc/gammu-smsd/gammu-smsdrc
else
    echo "警告: ATCONNECTION 环境变量未设置，使用默认配置"
fi

# 创建必要的目录
mkdir -p /data/log /data/db /var/log/gammu

# 设置目录权限
#chown -R gammu:gammu /data/log /data/db /var/log/gammu 2>/dev/null || true

# 如果数据库文件不存在，初始化数据库
if [ ! -f "/data/db/sms.db" ]; then
    echo "初始化 Gammu 数据库..."
    sqlite3 /data/db/sms.db < /etc/gammu-smsd/sqlite.sql
    # 设置数据库文件权限
#    chown gammu:gammu /data/db/sms.db 2>/dev/null || true
fi

# 检查转发脚本是否存在并具有执行权限
if [ -f "/usr/local/bin/forward-sms.sh" ]; then
    chmod +x /usr/local/bin/forward-sms.sh
    echo "短信转发脚本已就绪"

    # 设置转发脚本中的环境变量
    #if [ -n "$FORWARD_URL" ]; then
    #    sed -i "s|http://forwardsms:8080|$FORWARD_URL|g" /usr/local/bin/forward-sms.sh
    #fi
    #if [ -n "$FORWARD_SECRET" ]; then
    #    sed -i "s|your_shared_secret_here|$FORWARD_SECRET|g" /usr/local/bin/forward-sms.sh
    #fi
else
    echo "警告：未找到转发脚本 /usr/local/bin/forward-sms.sh"
fi

# 检查配置文件是否存在
if [ ! -f "/etc/gammu-smsd/gammu-smsdrc" ]; then
    echo "错误: /etc/gammu-smsd/gammu-smsdrc 配置文件不存在"
    exit 1
fi

# 显示配置信息
echo "=== Gammu SMSD 配置信息 ==="
echo "USB 端口: ${USB_PORT:-未设置}"
echo "转发 URL: ${FORWARD_URL:-未设置}"
echo "AT 连接: ${ATCONNECTION:-未设置}"
echo "数据库路径: /data/db/sms.db"
echo "日志路径: /data/log/"
echo "=========================="

# 执行传入的命令
exec "$@"