# DNSPod小助手 - 自动更新公网IP

[![Travis](https://img.shields.io/travis/rust-lang/rust.svg)]()

# 如何使用

1. 首先需要注册一个[DNSPod](https://www.dnspod.cn/)帐号，并将域名托管至改帐号下。

2. 在 账户管理 - 用户中心 - 安全设置 - `API Token` 处，创建一个Token。

3. 将`API_Token`的`ID`,`Token`填入配置文件。

4. 在`Records`中，填入需要更新的记录。

5. 完成配置，运行

  ```
  dnspodhelper -c 配置文件路径 - t 更新间隔时间(秒)
  ```

# 配置文件

```
{
  "Setting": {
    "api_token": {
      "id": "00001",
      "token": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    }
  },
  "Records": [
    {
      "domain": "域名",
      "sub_domain": "子域"
    },
    {
      "domain": "demo.com",
      "sub_domain": "demo2"
    }
  ]
}
```

# 更新记录

## v1.0 (2018/01/09 16:44:00)

- 发布第一版，支持多条记录更新，自动定时更新
