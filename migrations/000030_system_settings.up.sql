-- 000030_system_settings.up.sql
-- 系统设置（管理员自定义默认诊费，docs/20 S4 合并结算联动）
CREATE TABLE system_settings (
    key         VARCHAR(50)  PRIMARY KEY,          -- 设置键
    value       VARCHAR(200) NOT NULL DEFAULT '',  -- 设置值（金额单位：分）
    description VARCHAR(200),                      -- 说明
    updated_by  VARCHAR(50),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- 默认诊费：挂号费/诊查费（分）。CalculateBill 合并结算时若就诊无对应费用行且值>0，自动补一行。
INSERT INTO system_settings (key, value, description) VALUES
    ('default_registration_fee', '0',   '默认挂号费（分），合并结算自动带入'),
    ('default_consultation_fee', '0',   '默认诊查费（分），合并结算自动带入');

COMMENT ON TABLE system_settings IS '系统设置：管理员自定义默认诊费（000030）';
