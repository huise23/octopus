package op

import (
	"context"
	"fmt"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const dbDumpVersion = 1

func DBExportAll(ctx context.Context, includeLogs, includeStats bool) (*model.DBDump, error) {
	conn := db.GetDB().WithContext(ctx)

	d := &model.DBDump{
		Version:      dbDumpVersion,
		ExportedAt:   time.Now().UTC(),
		IncludeLogs:  includeLogs,
		IncludeStats: includeStats,
	}

	if err := conn.Find(&d.Channels).Error; err != nil {
		return nil, fmt.Errorf("export channels: %w", err)
	}
	if err := conn.Find(&d.Groups).Error; err != nil {
		return nil, fmt.Errorf("export groups: %w", err)
	}
	if err := conn.Find(&d.GroupItems).Error; err != nil {
		return nil, fmt.Errorf("export group_items: %w", err)
	}
	if err := conn.Find(&d.LLMInfos).Error; err != nil {
		return nil, fmt.Errorf("export llm_infos: %w", err)
	}
	if err := conn.Find(&d.APIKeys).Error; err != nil {
		return nil, fmt.Errorf("export api_keys: %w", err)
	}
	if err := conn.Find(&d.Settings).Error; err != nil {
		return nil, fmt.Errorf("export settings: %w", err)
	}
	if err := conn.Find(&d.SensitiveFilterRules).Error; err != nil {
		return nil, fmt.Errorf("export sensitive_filter_rules: %w", err)
	}

	if includeStats {
		if err := conn.Find(&d.StatsTotal).Error; err != nil {
			return nil, fmt.Errorf("export stats_total: %w", err)
		}
		if err := conn.Find(&d.StatsDaily).Error; err != nil {
			return nil, fmt.Errorf("export stats_daily: %w", err)
		}
		if err := conn.Find(&d.StatsHourly).Error; err != nil {
			return nil, fmt.Errorf("export stats_hourly: %w", err)
		}
		if err := conn.Find(&d.StatsModel).Error; err != nil {
			return nil, fmt.Errorf("export stats_model: %w", err)
		}
		if err := conn.Find(&d.StatsChannel).Error; err != nil {
			return nil, fmt.Errorf("export stats_channel: %w", err)
		}
		if err := conn.Find(&d.StatsAPIKey).Error; err != nil {
			return nil, fmt.Errorf("export stats_api_key: %w", err)
		}
	}

	if includeLogs {
		if err := conn.Find(&d.RelayLogs).Error; err != nil {
			return nil, fmt.Errorf("export relay_logs: %w", err)
		}
	}

	return d, nil
}

func DBImportIncremental(ctx context.Context, dump *model.DBDump) (*model.DBImportResult, error) {
	if dump == nil {
		return nil, fmt.Errorf("empty dump")
	}

	if dump.Version != 0 && dump.Version != dbDumpVersion {
		return nil, fmt.Errorf("unsupported dump version: %d", dump.Version)
	}

	conn := db.GetDB().WithContext(ctx)
	res := &model.DBImportResult{RowsAffected: map[string]int64{}}

	err := conn.Transaction(func(tx *gorm.DB) error {
		// base tables
		if n, err := createDoNothing(tx, dump.Channels); err != nil {
			return fmt.Errorf("import channels: %w", err)
		} else {
			res.RowsAffected["channels"] = n
		}
		if n, err := createDoNothing(tx, dump.Groups); err != nil {
			return fmt.Errorf("import groups: %w", err)
		} else {
			res.RowsAffected["groups"] = n
		}
		if n, err := createDoNothing(tx, dump.GroupItems); err != nil {
			return fmt.Errorf("import group_items: %w", err)
		} else {
			res.RowsAffected["group_items"] = n
		}
		if n, err := createUpsertAll(tx, dump.LLMInfos, []clause.Column{{Name: "name"}}); err != nil {
			return fmt.Errorf("import llm_infos: %w", err)
		} else {
			res.RowsAffected["llm_infos"] = n
		}
		if n, err := createDoNothing(tx, dump.APIKeys); err != nil {
			return fmt.Errorf("import api_keys: %w", err)
		} else {
			res.RowsAffected["api_keys"] = n
		}
		if n, err := createUpsertSettings(tx, dump.Settings); err != nil {
			return fmt.Errorf("import settings: %w", err)
		} else {
			res.RowsAffected["settings"] = n
		}

		// 分离内置规则和自定义规则
		var builtInRules []model.SensitiveFilterRule
		var customRules []model.SensitiveFilterRule
		for _, rule := range dump.SensitiveFilterRules {
			if rule.BuiltIn {
				builtInRules = append(builtInRules, rule)
			} else {
				customRules = append(customRules, rule)
			}
		}

		// 导入自定义规则（仅新增，不覆盖）
		if n, err := createDoNothing(tx, customRules); err != nil {
			return fmt.Errorf("import custom sensitive_filter_rules: %w", err)
		} else {
			res.RowsAffected["sensitive_filter_rules_custom"] = n
		}

		// 导入内置规则（仅更新 enabled 和 priority）
		if n, err := createUpsertBuiltInRules(tx, builtInRules); err != nil {
			return fmt.Errorf("import builtin sensitive_filter_rules: %w", err)
		} else {
			res.RowsAffected["sensitive_filter_rules_builtin"] = n
		}

		if dump.IncludeStats {
			if n, err := createUpsertAll(tx, dump.StatsTotal, []clause.Column{{Name: "id"}}); err != nil {
				return fmt.Errorf("import stats_total: %w", err)
			} else {
				res.RowsAffected["stats_total"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsDaily, []clause.Column{{Name: "date"}}); err != nil {
				return fmt.Errorf("import stats_daily: %w", err)
			} else {
				res.RowsAffected["stats_daily"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsHourly, []clause.Column{{Name: "hour"}}); err != nil {
				return fmt.Errorf("import stats_hourly: %w", err)
			} else {
				res.RowsAffected["stats_hourly"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsModel, []clause.Column{{Name: "id"}}); err != nil {
				return fmt.Errorf("import stats_model: %w", err)
			} else {
				res.RowsAffected["stats_model"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsChannel, []clause.Column{{Name: "channel_id"}}); err != nil {
				return fmt.Errorf("import stats_channel: %w", err)
			} else {
				res.RowsAffected["stats_channel"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsAPIKey, []clause.Column{{Name: "api_key_id"}}); err != nil {
				return fmt.Errorf("import stats_api_key: %w", err)
			} else {
				res.RowsAffected["stats_api_key"] = n
			}
		}

		if dump.IncludeLogs {
			if n, err := createDoNothing(tx, dump.RelayLogs); err != nil {
				return fmt.Errorf("import relay_logs: %w", err)
			} else {
				res.RowsAffected["relay_logs"] = n
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func createDoNothing[T any](tx *gorm.DB, rows []T) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	return result.RowsAffected, result.Error
}

func createUpsertAll[T any](tx *gorm.DB, rows []T, columns []clause.Column) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   columns,
		UpdateAll: true,
	}).Create(&rows)
	return result.RowsAffected, result.Error
}

func createUpsertSettings(tx *gorm.DB, rows []model.Setting) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&rows)
	return result.RowsAffected, result.Error
}

func createUpsertBuiltInRules(tx *gorm.DB, rows []model.SensitiveFilterRule) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	// 内置规则仅更新 enabled 和 priority 字段，不更新核心字段
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "priority"}),
	}).Create(&rows)
	return result.RowsAffected, result.Error
}
