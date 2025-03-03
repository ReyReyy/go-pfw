# 基于 Go 的端口转发工具

[English](docs.md) / 中文

### 目录:

- [配置文件](#配置文件)
- [全局参数](#全局参数)
- [服务参数](#服务参数)
- [网络](#网络)

## 配置文件

YAML 示例文件

```yaml
global:
  loglevel: info
  network:
    type: tcp
    send_proxy: false
    accept_proxy: false

services:
  - name: web
    listen: :80
    remote: example.com:80
    network:
      type: tcp
      send_proxy: true
```

<details>
  <summary>JSON 示例文件</summary>

```json
{
  "global": {
    "loglevel": "info",
    "network": [
      {
        "type": "tcp",
        "send_proxy": false,
        "accept_proxy": false
      }
    ]
  },
  "services": [
    {
      "name": "web",
      "listen": ":80",
      "remote": "example.com:80",
      "network": [
        {
          "type:": "tcp",
          "send_proxy": true
        }
      ]
    }
  ]
}
```

</details>

## 概览

```
├── golbal: Array
│   ├── loglevel: String
│   └── network: Array
└── services: Array
    ├── name: String
    ├── listen: String
    ├── remote: String
    └── network: Array
        ├── type: String | Array
        ├── send_proxy: Boolean
        └── accept_proxy: Boolean

```

## 全局参数

YAML 示例文件:

```yaml
global:
  loglevel: info
  network:
    type: tcp
    send_proxy: false
    accept_proxy: false
```

<details>
  <summary>JSON 示例文件:</summary>

```json
{
  "global": {
    "loglevel": "info",
    "network": [
      {
        "type": "tcp",
        "send_proxy": false,
        "accept_proxy": false
      }
    ]
  }
}
```

</details>

---

### `loglevel`: "none" | "info" | "debug"

定义日志等级。

- `none`: 关闭日志
- `info`: 开启普通日志
- `debug`: 开启除错日志

---

### `network`: Array

参考: [网络](#网络)。

## 服务参数

YAML 示例文件:

```yaml
services:
  service_name:
    listen: :8080
    remote: example:80
    network:
      type: tcp
      send_proxy: false
      accept_proxy: false

  another_service:
    listen: localhost:53
    remote: 8.8.8.8:53
    network:
      type: udp
```

<details>
  <summary>JSON 示例文件:</summary>

```json
{
  "services": {
    "service_name": {
      "listen": ":8080",
      "remote:": "example.com:80",
      "network": [
        {
          "type": "tcp",
          "send_proxy": false,
          "accept_proxy": false
        }
      ]
    },
    "another_service": {
      "listen": "localhost:53",
      "remote": "8.8.8.8:53",
      "network": [
        {
          "type": "udp"
        }
      ]
    }
  }
}
```

</details>
<br>

---

### `listen`: String

监听 IP 地址和端口。<br>
IP 可以留空，但是端口必须填写。<br>
如果 IP 留空，则会监听所有网卡。<br>

示例: `listen: :8080` 或 `listen: 127.0.0.1:53`<br>

---

### `remote`: String

远程 IP 地址和端口。<br>
IP 地址和端口都需要填写。<br>

示例: `remote: google.com:80` 或 `remote: 8.8.8.8:53`

---

### `network`: Array

参考: [网络](#网络)。

---

## 网络

```yaml
network:
  type: tcp
  send_proxy: false
  accept_proxy: false
```

<details>
  <summary>json</summary>

```json
{
  "network": [
    {
      "type": "tcp",
      "send_proxy": false,
      "accept_proxy": false
    }
  ]
}
```

</details>
<br>

> 如果[全局参数](#全局参数)和[服务参数](#服务参数)中都定义了[网络](网络)，则[服务参数](服务参数)中的[网络](网络)会覆盖全局参数。

---

### `type`: "tcp" | "udp" | "both" | Array

规定你想使用的网络类型<br>

- `tcp`: 仅使用 TCP 。
- `udp`: 仅使用 UDP 。
- `both`: 使用 TCP 和 UDP 两种类型。

> 你可以使用数组来定义多个网络类型，比如：
>
> ```yaml
> type: [tcp, udp]
> ```
>
> <details>
>   <summary>json</summary>
>
> ```json
> { "type": ["tcp", "udp"] }
> ```
>
> </details>

---

### `send_proxy`: true | false

布林值，用于控制是否发送
[HAProxy Protocol](https://www.haproxy.org/download/2.2/doc/proxy-protocol.txt)

你不能使用此选项于 UDP 协议。

---

### `accept_proxy`: true | false

布林值，用于控制是否接受
[HAProxy Protocol](https://www.haproxy.org/download/2.2/doc/proxy-protocol.txt)

你不能使用此选项于 UDP 协议。
