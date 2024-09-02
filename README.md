
## News Scrapy

虚拟币新闻资讯服务，提供数据抓取爬虫和数据展示接口服务。

#### 配置文件

配置文件采用`toml`格式语法配置

[`config.toml`](./config.toml)
```toml
# 接口服务配置
[api]
mode = "debug"  # 建议生产切换到release模式
addr = ":8080"

# 爬虫配置
[scrapy]
crontab = "@every 1h"
ua = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"

# Mysql数据库连接配置
[mysql]
host = "localhost"
port = 3306
username = "root"
password = "123456"
database = "news"

# Redis连接配置
[redis]
addr = "localhost:6379"
password = ""
db = 0

# Elasticsearch连接配置
[elastic]
addr = "http://localhost:9200"
index = "news-articles"

# Kimi AI配置
[kimi]
key = "sk-g3yv1wms1iZBdKZ61V0UEjN84W2PqZS49Gji2UCARy5AAa5L"
prompt = "你是一个专业的翻译员，用户输入的是中文则翻译成英文，用户输入的是其他语言则翻译成中文。如果输入的是单词则翻译结果需要小写开头，如果输入的是句子则翻译结果需要注意大小写。请你帮我逐字翻译"
```

#### Scrapy Tasks

任务文件：[`task.go`](./src/cmd/task.go)

爬虫实现方式：

- 采用[chromedp](https://github.com/chromedp/chromedp)库实现的模拟浏览器爬虫
- 采用[colly](https://go-colly.org/)库实现的http爬虫

爬虫调度执行时间配置

```toml
[scrapy]
crontab = "@every 1h"  # 每隔一小时执行一次
#crontab = "23 12 1 9 *" # 指定时间执行
```

#### API Server

接口文件：[`api.go`](./src/cmd/api.go)

启动爬虫任务和接口服务

```shell
go run src/main.go
```