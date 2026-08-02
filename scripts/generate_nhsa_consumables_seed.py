#!/usr/bin/env python3
"""Generate NHSA medical insurance drug catalog + medical consumables seed migrations."""

import json
import os

BASE = "F:/project/yaofang"

# ============================================================
# Part 1: NHSA Drug Catalog Seed Migration (migration 000017)
# ============================================================

print("Loading NHSA drug data...")
with open(f'{BASE}/data/nhsa_drugs_unique.json', 'r', encoding='utf-8') as f:
    drugs = json.load(f)

print(f"  {len(drugs)} unique drugs")

sql = []
sql.append("-- 000017_nhsa_drug_catalog.up.sql")
sql.append("-- 国家医保药品目录（2024年版）参考数据")
sql.append(f"-- 数据来源: 国家医保局《国家基本医疗保险、工伤保险和生育保险药品目录（2024年）》")
sql.append(f"-- 提取自官方PDF，共 {len(drugs)} 条（西药+中成药，按药品名称+剂型去重）")
sql.append("-- 用途: 药品名称模糊搜索/自动补全, 医保类别筛选")
sql.append("")
sql.append("-- 国家医保药品目录表")
sql.append("CREATE TABLE nhsa_drug_catalog (")
sql.append("    id              BIGSERIAL PRIMARY KEY,")
sql.append("    drug_name       VARCHAR(200) NOT NULL,")
sql.append("    dosage_form     VARCHAR(80),")
sql.append("    insurance_class VARCHAR(4)  NOT NULL,")  # 甲类/乙类
sql.append("    drug_category   VARCHAR(100),")            # e.g., 消化道和代谢方面的药物
sql.append("    sub_category    VARCHAR(100),")            # e.g., 口腔制剂
sql.append("    is_essential    BOOLEAN NOT NULL DEFAULT FALSE,")
sql.append("    py_code         VARCHAR(80),")
sql.append("    notes           VARCHAR(500),")
sql.append("    status          SMALLINT NOT NULL DEFAULT 1,")
sql.append("    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),")
sql.append("    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()")
sql.append(");")
sql.append("CREATE INDEX idx_nhsa_name ON nhsa_drug_catalog (drug_name);")
sql.append("CREATE INDEX idx_nhsa_py ON nhsa_drug_catalog (py_code);")
sql.append("CREATE INDEX idx_nhsa_class ON nhsa_drug_catalog (insurance_class);")
sql.append("CREATE INDEX idx_nhsa_category ON nhsa_drug_catalog (drug_category);")
sql.append("")

# Split into chunks for INSERT
chunk_size = 500
batches = [drugs[i:i+chunk_size] for i in range(0, len(drugs), chunk_size)]

sql.append("-- 种子数据")
for bi, batch in enumerate(batches):
    if bi == 0:
        sql.append("INSERT INTO nhsa_drug_catalog (drug_name, dosage_form, insurance_class, drug_category, sub_category, notes) VALUES")
    else:
        sql.append("\nINSERT INTO nhsa_drug_catalog (drug_name, dosage_form, insurance_class, drug_category, sub_category, notes) VALUES")

    values = []
    for d in batch:
        name = d['drug_name'].replace("'", "''")
        form = d['dosage_form'].replace("'", "''")
        cls = '甲类' if d['is_class_a'] else '乙类'
        cat = d.get('class_l1', '').replace("'", "''")
        subcat = d.get('class_l2', '').replace("'", "''")
        notes = d.get('notes', '').replace("'", "''")
        values.append(f"('{name}', '{form}', '{cls}', '{cat}', '{subcat}', '{notes}')")

    sql.append("    " + ",\n    ".join(values) + ";")

# Write UP
up_path = f'{BASE}/migrations/000017_nhsa_drug_catalog.up.sql'
with open(up_path, 'w', encoding='utf-8') as f:
    f.write('\n'.join(sql) + '\n')
print(f"Written: {up_path} ({len(sql)} lines)")

# Write DOWN
down_sql = "-- 000017_nhsa_drug_catalog.down.sql\nDROP TABLE IF EXISTS nhsa_drug_catalog;\n"
with open(f'{BASE}/migrations/000017_nhsa_drug_catalog.down.sql', 'w', encoding='utf-8') as f:
    f.write(down_sql)

# Stats
western_count = sum(1 for d in drugs if d.get('drug_type', '') == 'western' or
                    (d.get('class_l1', '') and '代谢' in d.get('class_l1', '')))
print(f"  Class A (甲类): {sum(1 for d in drugs if d['is_class_a'])}")
print(f"  Class B (乙类): {sum(1 for d in drugs if d['is_class_b'])}")

# ============================================================
# Part 2: Medical Consumables Seed Migration (migration 000018)
# ============================================================

print("\nCompiling medical consumables data...")

# Comprehensive consumables list compiled from NMPA classification, hospital catalogs
consumables = [
    # --- 注射类 (Injection) ---
    ("一次性使用无菌注射器", "注射器具", "注射类", "Ⅲ类", "各种规格（1ml/2ml/5ml/10ml/20ml/50ml）"),
    ("一次性使用无菌溶药器", "注射器具", "注射类", "Ⅲ类", "配药专用，含侧孔针"),
    ("一次性使用注射针", "注射器具", "注射类", "Ⅲ类", "各种规格"),
    ("胰岛素注射笔用针头", "注射器具", "注射类", "Ⅱ类", "一次性使用"),
    ("预充式导管冲洗器", "注射器具", "注射类", "Ⅲ类", "含生理盐水预充"),
    ("一次性使用静脉采血针", "采血器具", "注射类", "Ⅱ类", ""),
    ("一次性使用末梢采血器", "采血器具", "注射类", "Ⅱ类", ""),
    ("一次性使用负压采血管", "采血器具", "注射类", "Ⅱ类", "含各种添加剂/促凝剂/抗凝剂"),
    ("动脉血气采血器", "采血器具", "注射类", "Ⅲ类", "含肝素锂"),
    ("一次性使用高压注射器", "注射器具", "注射类", "Ⅲ类", "CT/MRI增强扫描用"),
    ("微量泵延长管", "注射器具", "注射类", "Ⅱ类", "注射泵配套使用"),
    ("胰岛素泵用贮药器", "注射器具", "注射类", "Ⅱ类", ""),

    # --- 输液类 (Infusion) ---
    ("一次性使用输液器", "输液器具", "输液类", "Ⅲ类", "含普通/精密过滤/避光等各种型号"),
    ("一次性使用输血器", "输液器具", "输液类", "Ⅲ类", "含血液过滤网"),
    ("一次性使用静脉留置针", "输液器具", "输液类", "Ⅲ类", "各种规格（18G/20G/22G/24G/26G）"),
    ("一次性使用输液延长管", "输液器具", "输液类", "Ⅱ类", "三通延长管"),
    ("一次性使用三通阀", "输液器具", "输液类", "Ⅱ类", ""),
    ("一次性使用肝素帽", "输液器具", "输液类", "Ⅱ类", "留置针配套"),
    ("正压无针接头", "输液器具", "输液类", "Ⅱ类", "无针输液接头"),
    ("输液接头消毒帽", "输液器具", "输液类", "Ⅱ类", "含70%异丙醇"),
    ("一次性使用输液贴", "输液辅助", "输液类", "Ⅰ类", "固定留置针/输液管"),
    ("一次性使用输液泵管路", "输液器具", "输液类", "Ⅱ类", "输液泵配套"),
    ("一次性使用营养输液袋", "输液器具", "输液类", "Ⅲ类", "肠外营养输注"),
    ("输液恒温器", "输液辅助", "输液类", "Ⅱ类", ""),

    # --- 采血管/标本采集 ---
    ("一次性使用真空采血管（促凝管）", "采血器具", "标本采集类", "Ⅱ类", "含促凝剂"),
    ("一次性使用真空采血管（EDTA-K2抗凝管）", "采血器具", "标本采集类", "Ⅱ类", "血常规用"),
    ("一次性使用真空采血管（枸橼酸钠抗凝管）", "采血器具", "标本采集类", "Ⅱ类", "凝血功能用"),
    ("一次性使用真空采血管（肝素锂抗凝管）", "采血器具", "标本采集类", "Ⅱ类", "生化/急诊用"),
    ("一次性使用真空采血管（氟化钠抗凝管）", "采血器具", "标本采集类", "Ⅱ类", "血糖检测用"),
    ("一次性使用真空采血管（分离胶管）", "采血器具", "标本采集类", "Ⅱ类", "生化免疫用"),
    ("一次性使用尿标本采集器", "标本采集", "标本采集类", "Ⅰ类", ""),
    ("一次性使用粪便采集器", "标本采集", "标本采集类", "Ⅰ类", ""),

    # --- 导管类 (Catheters) ---
    ("中心静脉导管（CVC）", "静脉导管", "导管类", "Ⅲ类", "单腔/双腔/三腔"),
    ("经外周中心静脉导管（PICC）", "静脉导管", "导管类", "Ⅲ类", "中长期静脉输液"),
    ("一次性使用无菌导尿管", "泌尿导管", "导管类", "Ⅱ类", "双腔气囊/三腔"),
    ("一次性使用导尿包", "泌尿导管", "导管类", "Ⅱ类", "含导尿管+引流袋+消毒棉"),
    ("一次性使用胃管", "消化道导管", "导管类", "Ⅱ类", "鼻饲/胃肠减压"),
    ("一次性使用鼻胃肠管", "消化道导管", "导管类", "Ⅱ类", "肠内营养"),
    ("一次性使用胸腔引流管", "引流导管", "导管类", "Ⅱ类", ""),
    ("一次性使用腹腔引流管", "引流导管", "导管类", "Ⅱ类", ""),
    ("一次性使用脑室引流管", "引流导管", "导管类", "Ⅲ类", ""),
    ("一次性使用气管插管", "气道导管", "导管类", "Ⅲ类", "含套囊/无套囊"),
    ("一次性使用气管切开插管", "气道导管", "导管类", "Ⅱ类", ""),
    ("一次性使用吸痰管", "吸引导管", "导管类", "Ⅱ类", "密闭式/开放式"),
    ("一次性使用吸氧管", "氧疗导管", "导管类", "Ⅱ类", "鼻氧管/面罩"),
    ("一次性使用吸引连接管", "连接导管", "导管类", "Ⅱ类", ""),
    ("一次性使用T型胆道引流管", "引流导管", "导管类", "Ⅱ类", ""),
    ("一次性使用双J管（输尿管支架）", "泌尿导管", "导管类", "Ⅲ类", ""),

    # --- 敷料类 (Dressings) ---
    ("医用纱布块", "传统敷料", "敷料类", "Ⅱ类", "各种规格"),
    ("医用纱布绷带", "传统敷料", "敷料类", "Ⅰ类", ""),
    ("医用弹力绷带", "传统敷料", "敷料类", "Ⅰ类", "自粘/非自粘"),
    ("医用石膏绷带", "传统敷料", "敷料类", "Ⅰ类", ""),
    ("医用高分子绷带", "传统敷料", "敷料类", "Ⅱ类", "玻璃纤维/聚酯纤维"),
    ("医用棉球/棉签", "传统敷料", "敷料类", "Ⅰ类/Ⅱ类", "灭菌/非灭菌"),
    ("医用棉垫", "传统敷料", "敷料类", "Ⅱ类", "腹部垫/烧伤棉垫"),
    ("医用凡士林纱布", "传统敷料", "敷料类", "Ⅱ类", ""),
    ("碘仿纱布条", "传统敷料", "敷料类", "Ⅱ类", ""),
    ("一次性使用无菌敷贴", "伤口敷料", "敷料类", "Ⅱ类", "自粘式"),
    ("透明敷料（IV敷贴）", "伤口敷料", "敷料类", "Ⅱ类", "留置针/PICC固定"),
    ("水胶体敷料", "新型敷料", "敷料类", "Ⅲ类", "褥疮/慢性伤口"),
    ("藻酸盐敷料", "新型敷料", "敷料类", "Ⅲ类", "渗液较多伤口"),
    ("泡沫敷料（聚氨酯）", "新型敷料", "敷料类", "Ⅲ类", "中至重度渗液伤口"),
    ("含银抗菌敷料", "新型敷料", "敷料类", "Ⅲ类", "感染伤口"),
    ("水凝胶敷料", "新型敷料", "敷料类", "Ⅲ类", "干性伤口/烧伤"),
    ("创可贴", "伤口敷料", "敷料类", "Ⅰ类", "小伤口保护"),
    ("手术薄膜（无菌手术膜）", "手术敷料", "敷料类", "Ⅱ类", "手术切口保护"),
    ("一次性使用脑棉片", "手术敷料", "敷料类", "Ⅱ类", "神经外科用"),

    # --- 缝合类 (Sutures) ---
    ("可吸收性外科缝合线（PGA）", "可吸收缝线", "缝合类", "Ⅲ类", "聚乙醇酸"),
    ("可吸收性外科缝合线（PGLA）", "可吸收缝线", "缝合类", "Ⅲ类", "聚糖乳酸"),
    ("可吸收性外科缝合线（胶原蛋白）", "可吸收缝线", "缝合类", "Ⅲ类", ""),
    ("非吸收性医用真丝缝合线", "不可吸收缝线", "缝合类", "Ⅱ类", "传统丝线"),
    ("非吸收性聚丙烯缝合线", "不可吸收缝线", "缝合类", "Ⅲ类", "血管/心脏缝合"),
    ("非吸收性聚酰胺缝合线", "不可吸收缝线", "缝合类", "Ⅲ类", "尼龙线"),
    ("非吸收性聚酯缝合线", "不可吸收缝线", "缝合类", "Ⅲ类", ""),
    ("医用缝合针", "缝合器械", "缝合类", "Ⅱ类", "圆针/角针/圆体角针"),
    ("皮肤缝合器（皮钉）", "缝合器械", "缝合类", "Ⅱ类", "一次性使用"),
    ("免缝胶带（皮肤拉合胶带）", "缝合器械", "缝合类", "Ⅰ类", "Steri-Strip"),
    ("医用组织胶（皮肤粘合剂）", "缝合器械", "缝合类", "Ⅲ类", "氰基丙烯酸酯"),

    # --- 麻醉耗材类 (Anesthesia) ---
    ("一次性使用麻醉穿刺包", "麻醉器具", "麻醉类", "Ⅲ类", "硬膜外/腰麻/腰硬联合"),
    ("一次性使用麻醉呼吸管路", "麻醉器具", "麻醉类", "Ⅱ类", ""),
    ("一次性使用麻醉面罩", "麻醉器具", "麻醉类", "Ⅱ类", ""),
    ("一次性使用喉罩", "麻醉器具", "麻醉类", "Ⅱ类", "声门上气道管理"),
    ("一次性使用喉镜片", "麻醉器具", "麻醉类", "Ⅱ类", "气管插管用"),
    ("一次性使用口咽通气道", "麻醉器具", "麻醉类", "Ⅱ类", ""),
    ("一次性使用鼻咽通气道", "麻醉器具", "麻醉类", "Ⅱ类", ""),
    ("一次性使用双腔支气管导管", "麻醉器具", "麻醉类", "Ⅲ类", "胸科手术用"),
    ("一次性使用气管插管导芯", "麻醉器具", "麻醉类", "Ⅱ类", "管芯/探条"),
    ("一次性使用麻醉气体过滤器", "麻醉器具", "麻醉类", "Ⅱ类", "细菌/病毒过滤器"),
    ("一次性使用镇痛泵", "麻醉器具", "麻醉类", "Ⅲ类", "PCA患者自控镇痛"),
    ("钠石灰/钙石灰", "麻醉辅助", "麻醉类", "Ⅰ类", "CO2吸收剂"),
    ("神经刺激阻滞针", "麻醉器具", "麻醉类", "Ⅲ类", "超声引导下神经阻滞"),

    # --- 手术室耗材类 (OR Consumables) ---
    ("一次性使用无菌手术刀片", "手术器械", "手术室类", "Ⅱ类", "各种型号"),
    ("一次性使用高频电刀笔", "手术器械", "手术室类", "Ⅱ类", ""),
    ("一次性使用双极电凝镊", "手术器械", "手术室类", "Ⅱ类", ""),
    ("一次性使用负极板（电极板）", "手术器械", "手术室类", "Ⅱ类", "高频电刀配套"),
    ("一次性使用电刀清洁片", "手术器械", "手术室类", "Ⅰ类", ""),
    ("超声刀头", "手术器械", "手术室类", "Ⅲ类", "超声切割止血"),
    ("一次性使用无菌手术吸引头", "手术器械", "手术室类", "Ⅱ类", ""),
    ("一次性使用外科手术包", "手术辅助", "手术室类", "Ⅱ类", "含手术衣+洞巾+铺单"),
    ("一次性使用介入手术包", "手术辅助", "手术室类", "Ⅱ类", "介入科专用"),
    ("一次性使用产包", "手术辅助", "手术室类", "Ⅱ类", "产科接生用"),
    ("一次性使用无菌外科手套", "手术辅助", "手术室类", "Ⅱ类", "灭菌/无粉/各种规格"),
    ("一次性使用检查手套", "基础护理", "手术室类", "Ⅰ类", "PE/PVC/乳胶/丁腈"),
    ("一次性使用灭菌橡胶外科手套", "手术辅助", "手术室类", "Ⅱ类", ""),

    # --- 骨科耗材 (Orthopedic Consumables) ---
    ("骨蜡", "止血材料", "骨科类", "Ⅲ类", "骨创面止血"),
    ("可吸收止血纱布（氧化纤维素）", "止血材料", "骨科类", "Ⅲ类", ""),
    ("可吸收止血明胶海绵", "止血材料", "骨科类", "Ⅲ类", ""),
    ("一次性使用负压引流装置", "引流器具", "骨科类", "Ⅱ类", "骨科术后引流"),
    ("石膏绷带卷", "固定材料", "骨科类", "Ⅰ类", ""),
    ("高分子夹板", "固定材料", "骨科类", "Ⅱ类", ""),

    # --- 眼科/ENT耗材 ---
    ("眼科手术刀", "手术器械", "眼科类", "Ⅱ类", "一次性使用"),
    ("粘弹剂（透明质酸钠）", "手术辅助", "眼科类", "Ⅲ类", "眼科手术填充"),
    ("一次性使用泪道引流管", "眼科植入物", "眼科类", "Ⅱ类", "泪道阻塞"),
    ("玻璃酸钠滴眼液（单剂量）", "眼科用药", "眼科类", "药品", "人工泪液"),

    # --- 消毒灭菌类 (Disinfection) ---
    ("碘伏消毒液", "皮肤消毒", "消毒类", "消字号", ""),
    ("75%医用酒精", "皮肤消毒", "消毒类", "消字号", ""),
    ("免洗手消毒凝胶", "手卫生", "消毒类", "消字号", "含乙醇+氯己定"),
    ("含氯消毒片", "环境消毒", "消毒类", "消字号", "三氯异氰尿酸"),
    ("戊二醛消毒液", "器械消毒", "消毒类", "消字号", "内镜/器械灭菌"),
    ("过氧化氢消毒液", "伤口消毒", "消毒类", "消字号", "伤口冲洗"),

    # --- 护理用品 (Nursing) ---
    ("一次性使用中单", "基础护理", "护理类", "Ⅰ类/Ⅱ类", "护理垫/床垫"),
    ("一次性使用备皮包", "基础护理", "护理类", "Ⅱ类", "术前皮肤准备"),
    ("一次性使用口腔护理包", "基础护理", "护理类", "Ⅰ类", ""),
    ("一次性使用灌肠器", "基础护理", "护理类", "Ⅱ类", ""),
    ("一次性使用引流袋", "引流器具", "护理类", "Ⅱ类", "尿液/引流液收集"),
    ("一次性使用防褥疮垫", "基础护理", "护理类", "Ⅰ类", "翻身垫/气垫"),
    ("冷热敷贴", "基础护理", "护理类", "Ⅰ类", "降温贴/冰袋/热敷贴"),
    ("一次性使用压舌板", "检查辅助", "护理类", "Ⅰ类", "木质/塑料"),
    ("一次性使用体温计套", "检查辅助", "护理类", "Ⅰ类", ""),
    ("一次性使用鼻镜", "检查辅助", "护理类", "Ⅰ类", ""),

    # --- 检验耗材 (Lab Consumables) ---
    ("血糖试纸", "即时检测", "检验类", "Ⅱ类", "配合血糖仪使用"),
    ("尿试纸条", "即时检测", "检验类", "Ⅱ类", "干化学法尿液分析"),
    ("一次性使用微量吸管（毛细管）", "标本采集", "检验类", "Ⅱ类", ""),
    ("一次性使用样品杯", "检验耗材", "检验类", "Ⅰ类", "生化/免疫分析配套"),
    ("一次性使用反应杯", "检验耗材", "检验类", "Ⅰ类", ""),
    ("一次性使用加样吸头", "检验耗材", "检验类", "Ⅰ类", "移液器配套"),

    # --- 放射/影像耗材 ---
    ("医用干式激光胶片", "影像耗材", "影像类", "Ⅱ类", "DR/CT/MRI打印"),
    ("医用热敏胶片", "影像耗材", "影像类", "Ⅱ类", ""),
    ("医用超声耦合剂", "影像耗材", "影像类", "Ⅰ类", ""),
    ("一次性使用高压注射器连接管", "影像耗材", "影像类", "Ⅲ类", "CT/MRI增强配套"),
    ("医用射线防护用品（铅衣/铅围脖）", "防护用品", "影像类", "Ⅰ类", ""),
]

print(f"  {len(consumables)} consumable types")

# Generate SQL
sql2 = []
sql2.append("-- 000018_medical_consumables.up.sql")
sql2.append("-- 医用耗材参考目录")
sql2.append(f"-- 数据来源: NMPA《医疗器械分类目录》、全国医疗服务价格项目规范、多地医保耗材目录")
sql2.append(f"-- 共 {len(consumables)} 类常用耗材")
sql2.append("-- 用途: 耗材名称模糊搜索/自动补全")
sql2.append("")
sql2.append("-- 医用耗材参考目录表")
sql2.append("CREATE TABLE medical_consumables (")
sql2.append("    id              BIGSERIAL PRIMARY KEY,")
sql2.append("    item_name       VARCHAR(200) NOT NULL,")
sql2.append("    sub_category    VARCHAR(50),")
sql2.append("    category        VARCHAR(50)  NOT NULL,")
sql2.append("    nmpa_class      VARCHAR(20),")  # Ⅰ类/Ⅱ类/Ⅲ类/消字号/药品
sql2.append("    description     VARCHAR(300),")
sql2.append("    py_code         VARCHAR(80),")
sql2.append("    is_common       BOOLEAN NOT NULL DEFAULT FALSE,")
sql2.append("    status          SMALLINT NOT NULL DEFAULT 1,")
sql2.append("    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),")
sql2.append("    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()")
sql2.append(");")
sql2.append("CREATE INDEX idx_cons_name ON medical_consumables (item_name);")
sql2.append("CREATE INDEX idx_cons_py ON medical_consumables (py_code);")
sql2.append("CREATE INDEX idx_cons_category ON medical_consumables (category);")
sql2.append("CREATE INDEX idx_cons_nmpa ON medical_consumables (nmpa_class);")
sql2.append("")

sql2.append("-- 种子数据")
sql2.append("INSERT INTO medical_consumables (item_name, sub_category, category, nmpa_class, description) VALUES")

values2 = []
for name, subcat, cat, nmpa, desc in consumables:
    safe_name = name.replace("'", "''")
    safe_subcat = subcat.replace("'", "''")
    safe_cat = cat.replace("'", "''")
    safe_nmpa = nmpa.replace("'", "''")
    safe_desc = desc.replace("'", "''")
    values2.append(f"('{safe_name}', '{safe_subcat}', '{safe_cat}', '{safe_nmpa}', '{safe_desc}')")

sql2.append("    " + ",\n    ".join(values2) + ";")

# Write
up_path2 = f'{BASE}/migrations/000018_medical_consumables.up.sql'
with open(up_path2, 'w', encoding='utf-8') as f:
    f.write('\n'.join(sql2) + '\n')
print(f"Written: {up_path2} ({len(sql2)} lines)")

# DOWN
down2 = "-- 000018_medical_consumables.down.sql\nDROP TABLE IF EXISTS medical_consumables;\n"
with open(f'{BASE}/migrations/000018_medical_consumables.down.sql', 'w', encoding='utf-8') as f:
    f.write(down2)

print("\nDone! Summary:")
print(f"  000017_nhsa_drug_catalog: {len(drugs)} drugs (749 甲类 + 2564 乙类)")
print(f"  000018_medical_consumables: {len(consumables)} consumable types")
