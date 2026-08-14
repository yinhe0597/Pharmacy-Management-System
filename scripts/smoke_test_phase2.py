#!/usr/bin/env python3
"""二期就诊模块冒烟：挂号→接诊→病历→处方发药→合并结算→收费。"""
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


# 1. 登录（admin）
r = call("POST", "/auth/login", {"username": "admin", "password": "admin123"})
check("登录", r)
TOKEN = r["data"]["token"]

# 0. 重置默认诊费为 0（保证结算明细口径确定：仅药费 1 行）
call("PUT", "/system-settings/default_registration_fee", {"value": "0"})
call("PUT", "/system-settings/default_consultation_fee", {"value": "0"})

# 2. 患者
r = call("POST", "/patients", {"card_no": "P2" + SUF, "name": "二期患者" + SUF, "gender": "男", "age": "45岁"})
check("建档", r)
pid = r["data"]["id"]

# 3. 挂号
r = call("POST", "/visits", {"patient_id": pid, "department": "内科", "visit_type": "outpatient"})
check("挂号", r)
vid = r["data"]["id"]
assert r["data"]["status"] == "waiting", r["data"]

# 4. 接诊
check("接诊", call("POST", f"/visits/{vid}/start"))
r = call("GET", f"/visits/{vid}")
assert r["data"]["status"] == "visiting", r["data"]
print("PASS 状态流转: waiting->visiting")

# 5. 病历
r = call("POST", f"/visits/{vid}/medical-record", {
    "chief_complaint": "咳嗽三天",
    "present_illness": "干咳无痰",
    "temperature": 37.5,
    "systolic_pressure": 120,
    "diastolic_pressure": 80,
    "pulse": 78,
    "diagnosis": "急性上呼吸道感染",
    "diagnosis_code": "A09",
    "diagnoses": [{"diagnosis_code": "A09", "diagnosis_name": "感染性腹泻", "is_primary": True}],
})
check("保存病历", r)
rec_id = r["data"]["id"]
r = call("GET", f"/visits/{vid}/medical-record")
assert r["data"]["id"] == rec_id and len(r["data"]["diagnoses"]) == 1
print("PASS 病历读取(含诊断):", r["data"]["diagnoses"][0]["diagnosis_code"])
# 6. 药品 + 处方（关联就诊）→ 发药
r = call("POST", "/drugs", {
    "code": "SMKV" + SUF, "generic_name": "二期测试药" + SUF, "dosage_form": "片剂",
    "specification": "0.5g", "manufacturer": "二期药厂",
    "base_unit": "盒", "split_unit": "片", "pack_size": 20,
    "is_split_allowed": True, "retail_price": 2000, "purchase_price": 1500,
})
check("新增药品", r)
did = r["data"]["id"]
r = call("POST", "/suppliers", {"code": "SMKVS" + SUF, "name": "二期供应商"})
supplier_id = r["data"]["id"]
r = call("POST", "/purchase-orders", {"supplier_id": supplier_id, "items": [
    {"drug_id": did, "quantity": 5, "unit_price": 1500}]})
po_id = r["data"]["id"]
call("POST", f"/purchase-orders/{po_id}/submit")
r = call("GET", f"/purchase-orders/{po_id}")
oi = r["data"]["items"][0]["id"]
r = call("POST", f"/purchase-orders/{po_id}/receive", {"items": [
    {"order_item_id": oi, "received_quantity": 5, "batch_no": "V2B1",
     "expiry_date": "2028-12-31T00:00:00+08:00", "qc_result": 1}]})
rid = r["data"]["id"]
check("采购收货", r)
check("确认入库", call("POST", f"/purchase-receipts/{rid}/complete"))
call("POST", "/inventory/transfer", {
    "from_location_id": 1, "to_location_id": 2,
    "items": [{"inventory_id": call("GET", f"/inventory?drug_id={did}&location_id=1")["data"]["list"][0]["id"], "quantity": 4}]})

# 拆零 1 盒 → 20 片（供处方拆零发药）
inv2 = [x for x in call("GET", f"/inventory?drug_id={did}&location_id=2")["data"]["list"] if not x["is_split"]][0]
check("拆零", call("POST", "/inventory/split", {"inventory_id": inv2["id"], "packs": 1}))

# 处方关联就诊并开立（visit_id）
r = call("POST", "/prescriptions", {
    "patient_id": pid, "patient_name": "二期患者" + SUF, "visit_id": vid,
    "source": "outpatient", "diagnosis_code": "A09",
    "items": [{"drug_id": did, "quantity": 12, "is_split": True,
               "single_dose": 1, "total_daily_dose": 3, "days": 4, "frequency": "tid"}]})
check("开立处方(关联就诊)", r)
rx_id = r["data"]["id"]
check("提交处方", call("POST", f"/prescriptions/{rx_id}/submit"))
check("审核通过", call("POST", f"/prescriptions/{rx_id}/review", {"action": "pass"}))
check("调配", call("POST", f"/prescriptions/{rx_id}/dispense"))
check("发药确认", call("POST", f"/prescriptions/{rx_id}/confirm-dispense", {"checker_id": 1}))

# 7. 结束就诊 → 合并结算
check("结束就诊", call("POST", f"/visits/{vid}/finish"))
r = call("POST", f"/visits/{vid}/charge", {"discount": 0})
check("生成合并结算单", r)
c_id = r["data"]["id"]
items = r["data"]["items"] if r["data"].get("items") else None
# items 在 Create 返回中被置空，重新查详情
r = call("GET", f"/charges/{c_id}")
assert len(r["data"]["items"]) == 1 and r["data"]["items"][0]["item_type"] == "drug", r["data"]
print("PASS 结算单明细:", [(i["item_name"], i["amount"]) for i in r["data"]["items"]])
assert r["data"]["payable_amount"] > 0
print("PASS 结算金额(分):", r["data"]["payable_amount"])

# 8. 收费
r = call("POST", f"/charges/{c_id}/pay", {"paid_amount": r["data"]["payable_amount"]})
check("收费", r)
r = call("GET", f"/charges/{c_id}")
assert r["data"]["status"] == "paid"
print("PASS 收费状态: pending->paid")

# 9. 退费
check("退费", call("POST", f"/charges/{c_id}/refund"))
r = call("GET", f"/charges/{c_id}")
assert r["data"]["status"] == "refunded"
print("PASS 退费状态: paid->refunded")

print("\n===== 二期就诊模块冒烟全部通过 =====")
