#!/usr/bin/env python3
"""管理员能力冒烟：默认诊费配置 + 用户管理（新建/改密） + 操作日志筛选。"""
import json
import sys
import time
import urllib.request

BASE = "http://localhost:8080/api/v1"
TOKEN = None
SUF = str(int(time.time()))[-6:]


def call(method, path, body=None):
    global TOKEN
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if TOKEN:
        req.add_header("Authorization", "Bearer " + TOKEN)
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        return json.loads(e.read().decode())


def check(name, r, expect_code=0):
    ok = r.get("code") == expect_code
    print(f"{'PASS' if ok else 'FAIL'} {name}: {r.get('message')}")
    if not ok:
        print("  ->", json.dumps(r, ensure_ascii=False)[:400])
        sys.exit(1)
    return r


# 1. 登录 admin
r = call("POST", "/auth/login", {"username": "admin", "password": "admin123"})
check("登录", r)
TOKEN = r["data"]["token"]

# 2. 系统设置：读取 + 更新默认诊费
r = call("GET", "/system-settings")
check("读取系统设置", r)
keys = {s["key"]: s["value"] for s in r["data"]}
assert "default_registration_fee" in keys and "default_consultation_fee" in keys
print("PASS 默认设置存在:", keys)
check("更新挂号费=500", call("PUT", "/system-settings/default_registration_fee", {"value": "500"}))
check("更新诊查费=2000", call("PUT", "/system-settings/default_consultation_fee", {"value": "2000"}))
r = call("GET", "/system-settings")
assert {s["key"]: s["value"] for s in r["data"]}["default_registration_fee"] == "500"
print("PASS 诊费已更新: registration=500, consultation=2000")
# 非法值校验
r = call("PUT", "/system-settings/default_consultation_fee", {"value": "-5"})
assert r.get("code") != 0
print("PASS 非法负值被拒绝")

# 3. 新建用户（财务）→ 修改密码 → 新密码登录
uname = "fin" + SUF
r = call("POST", "/users", {"username": uname, "password": "init123", "name": "测试财务", "role": "finance", "phone": "13800000000"})
check("新建财务用户", r)
uid = r["data"]["id"]
r = call("PUT", f"/users/{uid}", {"name": "测试财务", "role": "finance", "phone": "13800000000", "status": 1, "password": "reset456"})
check("重置密码", r)
r = call("POST", "/auth/login", {"username": uname, "password": "reset456"})
check("新密码登录", r)
r = call("POST", "/auth/login", {"username": uname, "password": "init123"})
assert r.get("code") != 0
print("PASS 旧密码已失效")

# 4. 操作日志：按用户/动作/时间筛选
r = call("GET", f"/operation-logs?user_id={uid}&page=1&page_size=10")
check("日志按用户筛选", r)
r = call("GET", "/operation-logs?action=create&page=1&page_size=10")
check("日志按动作筛选", r)
r = call("GET", "/operation-logs?action=create&start=2000-01-01T00:00:00%2B08:00&end=2100-01-01T23:59:59%2B08:00&page=1&page_size=10")
check("日志按时间窗口筛选", r)

# 5. 恢复默认诊费为 0（避免影响其他测试口径）
call("PUT", "/system-settings/default_registration_fee", {"value": "0"})
call("PUT", "/system-settings/default_consultation_fee", {"value": "0"})

print("\n===== 管理员能力冒烟全部通过 =====")
