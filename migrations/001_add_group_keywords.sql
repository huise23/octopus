-- 分组管理关键字匹配功能数据库迁移脚本
-- 版本: v1.0.0
-- 日期: 2024-12-24
-- 描述: 为 groups 表添加 keywords 和 match_mode 字段

-- 开始事务
BEGIN;

-- 添加新字段
ALTER TABLE groups ADD COLUMN keywords TEXT DEFAULT '';
ALTER TABLE groups ADD COLUMN match_mode INTEGER DEFAULT 0;

-- 为现有分组设置默认值
UPDATE groups SET keywords = '[]', match_mode = 0 WHERE keywords IS NULL OR keywords = '';

-- 添加注释
COMMENT ON COLUMN groups.keywords IS 'JSON格式的关键字列表，用于模型匹配';
COMMENT ON COLUMN groups.match_mode IS '匹配模式：0=仅分组名称，1=仅关键字，2=两者结合';

-- 添加索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_groups_match_mode ON groups(match_mode);

-- 验证数据完整性
DO $$
BEGIN
    -- 检查是否所有记录都有默认值
    IF EXISTS (SELECT 1 FROM groups WHERE keywords IS NULL OR match_mode IS NULL) THEN
        RAISE EXCEPTION '数据迁移失败：存在NULL值';
    END IF;

    -- 检查字段类型
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'groups'
        AND column_name = 'keywords'
        AND data_type = 'text'
    ) THEN
        RAISE EXCEPTION '字段类型验证失败：keywords字段类型不正确';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'groups'
        AND column_name = 'match_mode'
        AND data_type = 'integer'
    ) THEN
        RAISE EXCEPTION '字段类型验证失败：match_mode字段类型不正确';
    END IF;

    RAISE NOTICE '数据库迁移验证通过';
END $$;

-- 提交事务
COMMIT;

-- 迁移完成后的统计信息
SELECT
    COUNT(*) as total_groups,
    COUNT(CASE WHEN match_mode = 0 THEN 1 END) as name_only_groups,
    COUNT(CASE WHEN match_mode = 1 THEN 1 END) as keyword_only_groups,
    COUNT(CASE WHEN match_mode = 2 THEN 1 END) as both_mode_groups,
    COUNT(CASE WHEN keywords != '[]' AND keywords != '' THEN 1 END) as groups_with_keywords
FROM groups;

-- 显示迁移结果
\echo '数据库迁移完成！'
\echo '新增字段：'
\echo '  - keywords: TEXT类型，默认值为空数组[]'
\echo '  - match_mode: INTEGER类型，默认值为0（仅分组名称匹配）'
\echo '新增索引：'
\echo '  - idx_groups_match_mode: 优化匹配模式查询性能'