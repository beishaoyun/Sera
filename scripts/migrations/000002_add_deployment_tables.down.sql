-- 删除新增的 6 个表

DROP TABLE IF EXISTS scaling_policies;
DROP TABLE IF EXISTS natps_dlq_logs;
DROP TABLE IF EXISTS deployment_templates;
DROP TABLE IF EXISTS cloud_credentials;
DROP TABLE IF EXISTS ssh_recordings;
DROP TABLE IF EXISTS deployment_state_histories;

-- 删除 Deployment 表新增的字段
ALTER TABLE deployments DROP COLUMN IF EXISTS current_state;
ALTER TABLE deployments DROP COLUMN IF EXISTS state_history;
ALTER TABLE deployments DROP COLUMN IF EXISTS idempotency_key;
