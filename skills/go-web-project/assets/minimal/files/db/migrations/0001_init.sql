-- +goose Up
-- 骨架阶段的占位迁移：把 SELECT 1 换成你的第一张业务表。
-- 保留一条可执行语句，是为了让 make migrate-up / migrate-down 在空库上可验证。
SELECT 1;

-- +goose Down
SELECT 1;
