#!/bin/bash

set -e

sed -i "s|$USB_PORT|%USB_PORT%|g" /etc/gammu-smsd/gammu-smsdrc


# 如果数据库文件不存在，初始化数据库
if [ ! -f "/data/db/sms.db" ]; then
    echo "初始化 Gammu 数据库..."
    gammu-smsd -c /etc/gammu-smsd/gammu-smsdrc -s
fi

# 检查转发脚本是否存在并具有执行权限
if [ -f "/usr/local/bin/forward-sms.sh" ]; then
    chmod +x /usr/local/bin/forward-sms.sh
    echo "短信转发脚本已就绪"
else
    echo "警告：未找到转发脚本 /usr/local/bin/forward-sms.sh"
fi

# 执行传入的命令
exec "$@"