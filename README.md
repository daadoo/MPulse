# GFast-V3
<div align="center">
    <p>
        <h1>MPulse V1.0 服务器系统监控查看工具</h1>
    </p>
</div>

![star](https://gitcode.com/kocie/gfast/star/badge.svg)

## 工具简介
* 本项目(MPulse)由<a href="https://gitee.com/moses-studio" target="_blank"><b>Moses 愤怒的天天</b></a>开发。
* 基于全新Go开发服务器系统性能检查软件,查询所在服务器的CPU占用、内存占用、磁盘空间、高占用进程、数据库进程消耗、Nginx消耗、指定服务和端口消耗。
* 屏幕打印输出、日志输出
* 可以作为命令行工具，一次性查询当前系统信息
* 可以作为服务程序，以后台守护方式持续定时监控系统状态，默认web查询端口55055。
* 支持WEB接口查询系统信息结果。

需要gin版本后台管理 [https://github.com/qxkjsoft/ginfast](https://github.com/qxkjsoft/ginfast)

## 内置功能
* 查询系统CPU、内存、进程、磁盘使用情况    
* 查询 Nginx / 数据库 / 自定义服务 / 指定端口的进程资源消耗
* CLI 一次性采集、Daemon 常驻定时采集、HTTP JSON 接口

## 快速开始

### 构建
```bash
go build -o bin/mpulse ./cmd
```

### CLI 一次性采集
```bash
# 默认输出全部指标
./bin/mpulse -mode cli

# 指定端口与服务，只看 CPU Top 5
./bin/mpulse -mode cli -ports 80,443,3306 -services redis-server -top 5
```

### Daemon 常驻监控
```bash
./bin/mpulse -mode daemon -listen :55055 -interval 30 -log-file logs/mpulse.log
```

接口：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/health` | 存活与运行时长 |
| GET | `/api/v1/metrics` | 最近一轮采集结果（JSON） |
| GET | `/api/v1/metrics/refresh` | 立即触发一轮采集并返回 |

### 配置文件
复制 `config.example.yaml` 为 `config.yaml`，用 `-config config.yaml` 加载。
优先级：命令行参数 > 配置文件 > 默认值。

### 主要参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `-mode` | cli | cli / daemon |
| `-interval` | 30 | daemon 采集间隔（秒） |
| `-listen` | :55055 | HTTP 监听地址 |
| `-top` | 10 | Top 进程数量 |
| `-ports` | 空 | 观测端口，逗号分隔 |
| `-services` | 空 | 自定义服务进程名 |
| `-db-procs` | mysqld,mariadbd,postgres,mongod,redis-server | 数据库进程名 |
| `-nginx-procs` | nginx | Nginx 进程名 |
| `-ignore-proc` | 空 | 忽略的进程 |
| `-ignore-disk` | 空 | 忽略的挂载点 |
| `-log-level` | info | debug/info/warn/error |
| `-log-file` | 空 | 日志文件路径 |
| `-version` | - | 打印版本后退出 |

### 实现说明

- 采集层基于 `github.com/shirou/gopsutil/v4`，Windows 上可本地调试，Linux 上为单二进制部署。
- Web 层基于 `github.com/gin-gonic/gin`，只提供 JSON 接口，不加载前端资源。
- 一轮采集内各采集器并行执行，进程表只遍历一次并由所有采集器共享。
- 进程 CPU 采用「预热 + 150ms 窗口二次采样」，避免首轮返回 0 或历史均值。
- Windows 上 `SwapMemory` 走 WMI（单次约 2s），会拖慢监控自身，已只在 Linux/macOS 查询交换区。
- 单个采集器失败只记录到 `errors` 字段并写日志，不阻断整轮采集。
- 访问日志与采集日志统一走 `internal/logger`，`-log-level debug` 可看到每个采集器与每个 HTTP 请求的耗时。

## 配置
项目数据库文件 `resource/data/db.sql` 创建数据库导入后修改配置 `manifest/config/config.yaml.bak` 复制改为`manifest/config/config.yaml`

## 商用说明
> 商用注意事项下：
* 如果您在自己的项目中使用了我们项目中的扩展或模块，请在项目介绍中进行明确说明
