-- 000008_interaction_seed.up.sql
-- 药物相互作用规则种子数据（≥20条核心规则）。
-- 使用 ON CONFLICT DO NOTHING 保证幂等。

-- ============================================================
-- 1. 成分级相互作用规则
-- ============================================================
INSERT INTO ingredient_interactions (ingredient_a, ingredient_b, level, mechanism, evidence_level, description) VALUES
('华法林', '布洛芬',       1, 'NSAID抑制血小板聚集+胃黏膜损伤+蛋白结合置换华法林',        'A', '布洛芬增加华法林出血风险，禁忌联用'),
('华法林', '阿司匹林',     1, '协同抗凝+胃黏膜损伤',                                       'A', '阿司匹林与华法林禁忌联用，出血风险极大'),
('华法林', '对乙酰氨基酚', 2, '对乙酰氨基酚>2g/日可增强华法林抗凝作用',                    'B', '长期大量使用对乙酰氨基酚时需监测INR'),
('甲氨蝶呤', '布洛芬',     1, 'NSAID减少甲氨蝶呤肾清除，导致血药浓度升高',                  'A', '布洛芬与甲氨蝶呤禁忌联用，可致严重骨髓抑制'),
('甲氨蝶呤', '阿司匹林',   1, '水杨酸减少甲氨蝶呤肾清除',                                   'A', '阿司匹林与甲氨蝶呤禁忌联用'),
('庆大霉素', '呋塞米',     2, '袢利尿剂增强氨基糖苷类肾毒性',                               'B', '联用需监测肾功能和听力'),
('庆大霉素', '万古霉素',   2, '双重肾毒性+耳毒性叠加',                                     'B', '仅在必要时联用，严密监测肾功能'),
('地高辛', '呋塞米',       2, '呋塞米致低钾血症增加地高辛心脏毒性风险',                     'B', '联用需监测血钾和心电图'),
('卡马西平', '口服避孕药', 2, '卡马西平诱导CYP3A4加速避孕药代谢致避孕失败',                  'B', '建议改用其他避孕方式'),
('利福平', '华法林',       2, '利福平诱导CYP2C9加速华法林代谢降低抗凝效果',                  'B', '联用需监测INR调整华法林剂量'),
('克拉霉素', '辛伐他汀',   1, '克拉霉素抑制CYP3A4致辛伐他汀血药浓度升高，横纹肌溶解风险',    'A', '禁忌联用；改用阿奇霉素或普伐他汀'),
('酮康唑', '辛伐他汀',     1, '唑类抗真菌药强烈抑制CYP3A4',                                'A', '禁忌联用')
ON CONFLICT (ingredient_a, ingredient_b) DO NOTHING;

-- ============================================================
-- 2. 分类级相互作用规则
-- ============================================================
INSERT INTO class_interaction_rules (class_a, class_b, level, mechanism, evidence_level, description, is_active) VALUES
('NSAID',     'anticoagulant', 1, 'NSAID增强抗凝药作用+胃黏膜损伤+血小板抑制',            'A', 'NSAID与抗凝药禁忌联用', true),
('NSAID',     'antiplatelet',  2, '双重抗血小板作用增加消化道出血风险',                     'B', '需加用胃保护剂（PPI）', true),
('SSRI',      'MAOI',          1, '5-HT综合征风险：兴奋、高热、肌阵挛、甚至死亡',            'A', 'SSRI与MAOI禁忌联用；MAOI停用≥14天后方可启用SSRI', true),
('SSRI',      'SNRI',          2, '双重5-HT能作用增加5-HT综合征风险',                       'B', '慎用联用，注意观察5-HT综合征症状', true),
('MAOI',      'SNRI',          1, '5-HT综合征+高血压危象风险',                              'A', '禁忌联用', true),
('ACEI',      'ARB',           2, '双重RAS阻断增加低血压+高钾血症+肾损伤风险',               'B', '一般不推荐联用，仅在专科指导下', true),
('ACEI',      'spironolactone',2, '醛固酮拮抗剂+ACEI增加高钾血症风险',                      'B', '监测血钾', true),
('opioid',    'benzodiazepine',1, '协同呼吸抑制+镇静作用',                                  'A', '禁忌联用；仅在无替代方案且最低有效剂量下慎用', true),
('opioid',    'sedating',      2, '叠加镇静+呼吸抑制风险',                                 'B', '尽可能避免联用', true),
('anticholinergic', 'anticholinergic', 3, '抗胆碱能负荷叠加增加认知障碍、便秘、尿潴留风险', 'C', '老年人特别注意抗胆碱能负荷', true),
('QT-prolonging', 'QT-prolonging', 2, 'QT间期叠加延长增加尖端扭转型室速（TdP）风险',       'B', '联用需做心电图监测', true)
ON CONFLICT (class_a, class_b) DO NOTHING;

-- ============================================================
-- 3. 标签级相互作用规则
-- ============================================================
INSERT INTO tag_interactions (tag_a, tag_b, level, mechanism, evidence_level, description, is_active) VALUES
('cyp3a4-inhibitor', 'cyp3a4-substrate', 2, '抑制CYP3A4代谢致底物药物血药浓度升高',    'B', '需调整底物药物剂量或更换药物', true),
('cyp3a4-inducer',   'cyp3a4-substrate', 2, '诱导CYP3A4代谢致底物药物血药浓度降低',    'B', '需增加底物药物剂量或更换药物', true),
('cyp2d6-inhibitor', 'cyp2d6-substrate', 2, '抑制CYP2D6代谢',                          'B', '监测底物药物疗效/毒性', true),
('serotonergic',     'serotonergic',     2, '双重5-HT能作用增加5-HT综合征风险',          'B', '监测5-HT综合征症状', true),
('qt-prolonging',    'cyp3a4-inhibitor', 2, 'CYP3A4抑制剂增加QT延长药物血药浓度',        'C', '监测QTc间期', true),
('nsaid',            'anticoagulant',    1, 'NSAID与抗凝药标签交互',                     'A', '同分类级规则，提供标签视角冗余', true),
('nsaid',            'nephrotoxic',      2, 'NSAID减少肾灌注可能加重肾毒性药物损伤',      'B', '老年/肾功能不全者慎用', true),
('sedating',         'sedating',         3, '双重镇静作用叠加',                          'C', '警告患者避免驾驶和操作机械', true)
ON CONFLICT (tag_a, tag_b) DO NOTHING;

-- ============================================================
-- 4. 患者禁忌规则（示例）
-- ============================================================
INSERT INTO patient_contraindications (drug_id, ingredient, contraindication_type, condition_value, level, description) VALUES
(NULL, '阿司匹林', 'age',      '<12',   1, '阿司匹林在儿童中可致Reye综合征（病毒感染时）'),
(NULL, '四环素',   'age',      '<8',    1, '四环素影响儿童骨骼和牙齿发育'),
(NULL, '庆大霉素', 'age',      '<6',    2, '氨基糖苷类在婴幼儿中慎用'),
(NULL, '布洛芬',   'pregnancy','3rd',   1, '布洛芬在妊娠晚期可致动脉导管早闭'),
(NULL, '利巴韦林', 'pregnancy','all',   1, '利巴韦林有明确致畸性，妊娠期禁用'),
(NULL, '甲硝唑',   'lactation','all',   2, '甲硝唑经乳汁排泄，哺乳期慎用')
ON CONFLICT DO NOTHING;
