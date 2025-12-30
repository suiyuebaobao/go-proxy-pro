# OSINT 信息收集工具大全

> 整理自聊天记录，仅供安全研究、渗透测试、CTF等合法用途使用

---

## 一、域名/DNS 信息查询

| 网站 | 地址 | 说明 |
|------|------|------|
| SecurityTrails | https://securitytrails.com | 最全的DNS历史记录，需注册 |
| ViewDNS | https://viewdns.info/iphistory/ | 免费IP历史查询 |
| DNS History | https://dnshistory.org | DNS历史记录 |
| CompleteDNS | https://completedns.com/dns-history/ | DNS历史记录 |
| WhoisRequest | https://whoisrequest.com/history/ | Whois+DNS历史 |
| Whois查询 | https://who.is | 域名Whois信息 |

---

## 二、SSL证书搜索

| 网站 | 地址 | 说明 |
|------|------|------|
| Censys | https://search.censys.io | 证书+IP搜索，功能强大 |
| crt.sh | https://crt.sh | 证书透明度日志查询 |
| Shodan | https://www.shodan.io | 设备/服务搜索引擎 |
| FOFA | https://fofa.info | 国内版Shodan，资产搜索 |
| Hunter | https://hunter.io | 资产搜索+邮箱发现 |

---

## 三、子域名枚举

| 网站 | 地址 | 说明 |
|------|------|------|
| DNSDumpster | https://dnsdumpster.com | 免费，可视化展示 |
| Subdomainfinder | https://subdomainfinder.c99.nl | 子域名查询 |
| Phonebook.cz | https://phonebook.cz | 子域名+邮箱发现 |
| Chaos | https://chaos.projectdiscovery.io | 海量子域名数据 |
| HackerTarget | https://api.hackertarget.com/hostsearch/?q=域名 | 免费API |

---

## 四、IP 信息查询

| 网站 | 地址 | 说明 |
|------|------|------|
| IP-API | http://ip-api.com/json/IP地址 | 免费IP归属地API |
| IPInfo | https://ipinfo.io/IP地址 | IP详细信息 |
| BGP.HE.NET | https://bgp.he.net | BGP/IP/ASN信息 |
| AbuseIPDB | https://www.abuseipdb.com | IP信誉/举报查询 |

---

## 五、综合OSINT平台

| 网站 | 地址 | 说明 |
|------|------|------|
| IntelX | https://intelx.io | 综合情报搜索引擎 |
| OSINT Framework | https://osintframework.com | OSINT工具导航大全 |
| Maltego | https://www.maltego.com | 专业情报分析工具 |
| Spiderfoot | https://www.spiderfoot.net | 自动化OSINT工具 |

---

## 六、泄露数据查询

| 网站 | 地址 | 说明 |
|------|------|------|
| Have I Been Pwned | https://haveibeenpwned.com | 查邮箱是否泄露 |
| DeHashed | https://dehashed.com | 泄露数据搜索（付费） |
| LeakCheck | https://leakcheck.io | 泄露数据查询 |

---

## 七、手机号/社交账号

| 网站 | 地址 | 说明 |
|------|------|------|
| IP138手机归属地 | https://www.ip138.com/sj/ | 手机号归属地查询 |
| 电话查 | https://www.dianhuacha.com | 号码归属+标记查询 |
| Namechk | https://namechk.com | 用户名跨平台查询 |
| WhatsMyName | https://whatsmyname.app | 用户名全网搜索 |

---

## 八、GitHub 信息

| 地址 | 说明 |
|------|------|
| https://api.github.com/users/用户名 | 查询GitHub用户信息 |
| https://api.github.com/search/users?q=关键词 | 搜索GitHub用户 |
| https://api.github.com/search/repositories?q=关键词 | 搜索仓库 |

---

## 九、命令行工具

```bash
# Whois查询
whois example.com

# DNS查询
dig example.com ANY +noall +answer
nslookup example.com

# 子域名枚举
subfinder -d example.com

# 端口扫描
nmap -sn 192.168.1.0/24

# HTTP探测
echo "example.com" | httpx -ip

# 证书查询
echo | openssl s_client -connect example.com:443 2>/dev/null | openssl x509 -noout -text
```

---

## 十、Cloudflare 相关

### CF真实IP发现思路

1. **历史DNS记录** - SecurityTrails, ViewDNS
2. **子域名泄露** - 某些子域可能没走CF
3. **邮件头** - 发件服务器可能暴露真实IP
4. **SSL证书** - Censys搜索证书关联IP
5. **同IP其他站点** - 反查同服务器域名

### CF IP段（配置防火墙用）

```
https://www.cloudflare.com/ips-v4
https://www.cloudflare.com/ips-v6
```

---

## 十一、防护建议

### 源站防护

```bash
# 只允许CF的IP访问80/443
for ip in $(curl -s https://www.cloudflare.com/ips-v4); do
  iptables -A INPUT -p tcp -s $ip --dport 443 -j ACCEPT
  iptables -A INPUT -p tcp -s $ip --dport 80 -j ACCEPT
done
iptables -A INPUT -p tcp --dport 443 -j DROP
iptables -A INPUT -p tcp --dport 80 -j DROP
```

### Nginx限流

```nginx
limit_req_zone $binary_remote_addr zone=one:10m rate=10r/s;
limit_conn_zone $binary_remote_addr zone=addr:10m;

server {
    limit_req zone=one burst=20 nodelay;
    limit_conn addr 10;
}
```

---

## 免责声明

本文档仅供以下用途：
- 安全研究与学习
- 授权的渗透测试
- CTF竞赛
- 自身资产的安全检测

**严禁用于非法入侵、隐私侵犯等违法活动！**

---

> 整理时间：2025-12-12
> 来源：Claude Code 聊天整理
