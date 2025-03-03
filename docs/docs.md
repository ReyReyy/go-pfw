# Go Port Forwarding Tool Documentation

English / [中文](docs_CN.md)

### directory:

- [Configuration File](#configuration-file)
- [Global Options](#global-options)
- [Services Options](#services-options)
- [Network](#network)

## Configuration File

YAML Example

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
  <summary>JSON Example</summary>

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

## Overview

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

## Global Options

YAML Example:

```yaml
global:
  loglevel: info
  network:
    type: tcp
    send_proxy: false
    accept_proxy: false
```

<details>
  <summary>JSON Example:</summary>

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

Specify what you wanna see in logs.

- `none`: Disable logging
- `info`: Info level logging
- `debug`: Debug level logging

---

### `network`: Array

Read: [Network](#network)

## Services Options

YAML Example:

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
  <summary>JSON Example:</summary>

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

Listen address and port.<br>
IP can be empty but Port is required.<br>
If IP is empty, it will bind to all interfaces.<br>

Example: `listen: :8080` or `listen: 127.0.0.1:53`<br>

---

### `remote`: String

Remote address and port you want port forward to.<br>
IP and Port both are required.<br>

Example: `remote: google.com:80` or `remote: 8.8.8.8:53`

---

### `network`: Array

Read: [Network](#network).

---

## Network

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

> If the `network` is set in both `global` and `services`, the `service`'s network will override the `global`'s.

---

### `type`: "tcp" | "udp" | "both" | Array

Specifies the type of network protocol to use.<br>

- `tcp`: Use TCP only.
- `udp`: Use UDP only.
- `both`: Use both TCP and UDP.

> You can also use an array to specify multiple types, e.g.,
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

Boolean options to control whether to send
[HAProxy Protocol](https://www.haproxy.org/download/2.2/doc/proxy-protocol.txt)

You can't use this option with udp protocol

---

### `accept_proxy`: true | false

Boolean options to control whether to accept
[HAProxy Protocol](https://www.haproxy.org/download/2.2/doc/proxy-protocol.txt)

You can't use this option with udp protocol
