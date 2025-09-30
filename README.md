# 项目介绍

修改自[ForwardSMS](https://github.com/SecurityPaper/ForwardSMS)


4G模块，sim卡短信转发到企业微信软硬一体解决方案

企业过难免遇到注册外部账号情况，如果使用个人手机号很容易离职时候导致账号难以管理。

所以使用公司公共手机号和电话卡，比如A业务需要注册外部手机号，

可以直接拉群然后使用企业微信群机器人功能，把收到短信进行关键字监控，自动转发到群里。

如果B业务也需要用外部手机号，可以根据B业务的短信模板监控B业务的关键字。

自动转发短信到B业务的群里。

解决人员离职手机号不能归属企业问题。

## 为什么写了这么个项目

最开始准备采用大神pppscn写的[SmsForwarder](https://github.com/pppscn/SmsForwarder)

但是安卓保活太麻烦了，比较好的解决方案就是插着电脑，不断用adb命令调起程序。

而且锂电池太容易炸了，放在机房有点不放心。

所以实现了这么一个方案，方便放在机房或者插在树莓派上直接使用

## 支持CPU架构

* amd64
* arm64
## 使用硬件

使用移远EC20解决方案，[购买链接](https://detail.tmall.com/item.htm?spm=a1z10.5-b-s.w4011-23773508522.66.cd38a48eir4WR3&id=595437612613&skuId=4304510502236)

## EC 模块初始化和配置
将含有模块的转接板插到 USB 口之后使用命令 `ls /dev/ttyUSB*` 查看当前系统中的 tty 模拟 usb 的设备，如果当前机器只有 ec20 模块的话大概会看到 `ttyUSB1`/`ttyUSB2`/`ttyUSB3`, 有些还会有 `ttyUSB4` 甚至更高，根据移远文档的说明，通常 `ttyUSB2` 是 `AT` 命令口，这也是本文主要用到的接口.


确定转接板链接正常后通过 `apt install minicom` 安装 `minicom` 然后通过命令 `minicom -D /dev/ttyUSB2` 连接到模块执行 `AT` 命令
输入 `ATI` 按回车可以看到模块返回的模块信息，大概如下

```shell
Quectel
EC20F
Revision: EC20CEFAGR06A10M4G
```

需要注意的是有些模块可能没有 `ttyUSB2`, 但是有 `ttyUSB3` 和 `ttyUSB4`, 这时可以尝试使用 `ttyUSB3`.
如果设备插了多个模块和转接板，则后面的数字是顺延排列的，所以其实叫什么名无所谓，可以全都发一下 `AT` 指令试一下.


然后输入 `AT+QPRTPARA=3` 重置模块设置，然后输入 `AT+CFUN=1,1` 重启模块，重启时模块会和机器短暂断开连接，如果长时间链接不上可以执行 `ls /dev/ttyUSB*` 看下数字是否发生变化，有时数字会往后顺延，这时需要重新拔插 usb 或者直接连接新的接口.


然后输入 `AT+COPS?` 和 `AT+QNWINFO` 查询是否已经注册到运营商网络，大概会返回如下信息

```shell
AT+COPS?
+COPS: 0,0,"CHN-UNICOM",7

OK

AT+QNWINFO
+QNWINFO: "FDD LTE","46001","LTE BAND 3",1650

OK
```
到这里 EC 模块就算是配置好了，如果没有你的 SIM 卡没有注册成功的话，先排除是否是欠费了，没有的话可以尝试开启 VOLTE, 操作方式的话还请 google 搜索.


## 部署方案

1. 插入sim卡
2. 插入电脑usb口
3. 如果指示灯蓝绿交替常亮，代表识别卡信号成功，如果交替闪烁，代表没信号，请安装天线后尝试
4. 使用`lsusb`命令查看是否识别成功
5. 克隆项目到本地 `git clone https://github.com/scjtqs2/ForwardSMS.git && cd ForwardSMS`
6. 注意，当前目录必须为项目内目录，然后使用命令 `docker run --privileged -v /dev/ttyUSB3:/dev/ttyUSB3 -v ./data:/data securitypaperorg/gammu-smsd:latest echo "a test sms from ec20" | /usr/bin/gammu -c /data/config/gammu-smsd.conf sendsms TEXT 133xxxxxxx`
7. 133xxxxxxx请替换为自己手机号
8. 如果成功发送短信，代表卡识别正确，如果返回`350`代表卡并未搜到信号。如果长时间没反应，请尝试另外几个`/dev/ttyUSBx`,直到发送短信成功。
9. 短信发送成功后配置`data/config/forward.yaml`文件
10. 根据第6条测试出来的usb端口号，配置`docker-compose.yaml`文件
11. 在当前目录执行`docker-compose up -d`

## 文件解释

`data/config/forward.yaml`
```yaml
# 如果有all这个配置，就是默认所有短信都会转发给这个机器人，建议发送给管理员，或者直接删除关闭
all:
  rule: all
  type: all
  url: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxxxxx

# 从上到下依次为 项目名称、规则（使用关键字匹配）、匹配方式（后续可能支持正则）、机器人url
测试:
  rule: 测试
  type: keyword
  url: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxxxxx
```

---

`data/config/gammu-smsd.conf`

```conf
[gammu]
# 配置短信转发端口，映射为USB3，但是要看清楚主机的端口地址，测试端口请查看文档
port = /dev/ttyUSB3

#连接方式和速率，保持默认即可
connection = at115200

# 配置smsd守护进程的配置
[smsd]

# 使用的存档模式
Service = sql

# 使用具体的数据库
Driver = sqlite3

# 数据库路径
DBDir = /data/db

# 数据库名称
Database = sms.db

# 日志存放位置
logfile = /data/log/gammu-smsd.log

# 开启debug，默认不开启
debuglevel = 0

```
---

---
`data/db/sms.db`
> 文件为sqlite3数据库，用来存储短信接收，如果有需要请定期备份，或者可以下载后查看。

---
`data/log/gammu-smsd.log`
> 这个文件是gammu-smsd服务产生的日志