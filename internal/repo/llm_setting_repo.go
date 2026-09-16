package repo

import (
	"customerservicecore/internal/model"

	"gorm.io/gorm"
)

type LlmSettingRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewLlmSettingRepo(db *gorm.DB) *LlmSettingRepo {
	return &LlmSettingRepo{db: db}
}

func (r *LlmSettingRepo) ForTenant(tenantID uint64) *LlmSettingRepo {
	return &LlmSettingRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *LlmSettingRepo) GetOrDefault() (*model.CsLlmSetting, error) {
	var row model.CsLlmSetting
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&row).Error
	if err == nil {
		return &row, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return &model.CsLlmSetting{
		TenantID:    r.tenantID,
		Enabled:     false,
		CooldownSec: 25,
	}, nil
}

func (r *LlmSettingRepo) Save(row *model.CsLlmSetting) error {
	row.TenantID = r.tenantID
	if row.ID == 0 {
		var existing model.CsLlmSetting
		err := r.db.Scopes(scopeTenant(r.tenantID)).First(&existing).Error
		if err == nil {
			row.ID = existing.ID
			row.CreatedAt = existing.CreatedAt
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
	}
	return r.db.Save(row).Error
}
