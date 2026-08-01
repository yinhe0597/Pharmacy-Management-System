#!/usr/bin/env python3
"""HTTP 全链路冒烟测试：登录→药品→供应商→采购→入库→处方→发药→退药。"""
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
            r = json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        r = json.loads(e.read().decode())
    return r


def check(name, r, expect_code=0):
    ok = r.get("code") == expect_code
    print(f"{'PASS' if ok else 'FAIL'} {name}: {r.get('message')}")
    if not ok:
        print("  ->", json.dumps(r, ensure_ascii=False)[:400])
        sys.exit(1)
    return r


# 1. 登录
r = call("POST", "/auth/login", {"username": "admin", "password": "admin123"})
check("登录", r)
TOKEN = r["data"]["token"]

# 2. 药品
r = call("POST", "/drugs", {
    "code": "SMK" + SUF, "generic_name": "冒烟测试药" + SUF, "dosage_form": "片剂",
    "specification": "0.5g", "manufacturer": "冒烟药厂",
    "base_unit": "盒", "split_unit": "片", "pack_size": 20,
    "is_split_allowed": True, "retail_price": 2000, "purchase_price": 1500,
})
check("新增药品", r)
drug_id = r["data"]["id"]

# 3. 供应商
r = call("POST", "/suppliers", {"code": "SMKS" + SUF, "name": "冒烟供应商"})
check("新增供应商", r)
supplier_id = r["data"]["id"]

# 4. 采购单 → 提交 → 收货 → 确认入库
r = call("POST", "/purchase-orders", {"supplier_id": supplier_id, "items": [
    {"drug_id": drug_id, "quantity": 5, "unit_price": 1500}]})
check("创建采购单", r)
po_id = r["data"]["id"]
check("提交采购单", call("POST", f"/purchase-orders/{po_id}/submit"))
# 收货需要 order_item_id，从采购单详情获取
r = call("GET", f"/purchase-orders/{po_id}")
order_item_id = r["data"]["items"][0]["id"]
r = call("POST", f"/purchase-orders/{po_id}/receive", {"items": [
    {"order_item_id": order_item_id, "received_quantity": 5, "batch_no": "SMKB1",
     "expiry_date": "2028-12-31T00:00:00+08:00", "qc_result": 1}]})
check("采购收货", r)
receipt_id = r["data"]["id"]
check("确认入库", call("POST", f"/purchase-receipts/{receipt_id}/complete"))

# 5. 调拨 4 盒从药库到药房
r = call("GET", f"/inventory?drug_id={drug_id}&location_id=1")
inv1 = [x for x in r["data"]["list"] if not x["is_split"]][0]
check("调拨到药房", call("POST", "/inventory/transfer", {
    "from_location_id": 1, "to_location_id": 2,
    "items": [{"inventory_id": inv1["id"], "quantity": 4}]}))

# 6. 拆零 2 盒 → 40 片
r = call("GET", f"/inventory?drug_id={drug_id}&location_id=2")
inv = [x for x in r["data"]["list"] if not x["is_split"]][0]
check("拆零", call("POST", "/inventory/split", {"inventory_id": inv["id"], "packs": 2}))

# 6. 处方：拆零 12 片
r = call("POST", "/prescriptions", {
    "patient_name": "冒烟患者" + SUF, "patient_age": "30岁",
    "items": [{"drug_id": drug_id, "quantity": 12, "is_split": True,
               "single_dose": 1, "total_daily_dose": 3, "days": 4, "frequency": "tid"}]})
check("创建处方", r)
rx_id = r["data"]["id"]
check("提交处方", call("POST", f"/prescriptions/{rx_id}/submit"))
check("审核通过", call("POST", f"/prescriptions/{rx_id}/review", {"action": "pass"}))
check("调配", call("POST", f"/prescriptions/{rx_id}/dispense"))
check("发药确认", call("POST", f"/prescriptions/{rx_id}/confirm-dispense", {"checker_id": 1}))

# 7. 检查库存：拆零行应为 40-12=28 片
r = call("GET", f"/inventory?drug_id={drug_id}&location_id=2")
split_qty = [x for x in r["data"]["list"] if x["is_split"]][0]["quantity"]
assert split_qty == 28, f"拆零库存应为28, got {split_qty}"
print(f"PASS 库存校验: 拆零剩余 {split_qty} 片")

# 8. 退药 4 片 → 32 片
r = call("GET", f"/prescriptions/{rx_id}")
item_id = r["data"]["items"][0]["id"]
check("退药", call("POST", f"/prescriptions/{rx_id}/return", {"items": [
    {"item_id": item_id, "return_quantity": 4}]}))
r = call("GET", f"/inventory?drug_id={drug_id}&location_id=2")
split_qty = [x for x in r["data"]["list"] if x["is_split"]][0]["quantity"]
assert split_qty == 32, f"退药后拆零库存应为32, got {split_qty}"
print(f"PASS 退药校验: 拆零库存回补至 {split_qty} 片")

# 9. 报表
r = call("GET", "/reports/dispensing-workload?start=2026-08-01T00:00:00%2B08:00&end=2026-08-01T23:59:59%2B08:00")
print("PASS 调配工作量报表:", r.get("data"))

print("\n===== 冒烟测试全部通过 =====")
