package repo

import (
	"customerservicecore/internal/model"

	"gorm.io/gorm"
)

type AutoReplyRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewAutoReplyRepo(db *gorm.DB) *AutoReplyRepo {
	return &AutoReplyRepo{db: db}
}

func (r *AutoReplyRepo) ForTenant(tenantID uint64) *AutoReplyRepo {
	return &AutoReplyRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *AutoReplyRepo) List() ([]model.CsAutoReplyRule, error) {
	var list []model.CsAutoReplyRule
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Order("priority ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *AutoReplyRepo) ListEnabledForShop(shopID uint64) ([]model.CsAutoReplyRule, error) {
	var list []model.CsAutoReplyRule
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("enabled = ? AND (shop_id = 0 OR shop_id = ?)", true, shopID).
		Order("priority ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *AutoReplyRepo) Get(id uint64) (*model.CsAutoReplyRule, error) {
	var row model.CsAutoReplyRule
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AutoReplyRepo) Create(row *model.CsAutoReplyRule) error {
	row.TenantID = r.tenantID
	return r.db.Create(row).Error
}

func (r *AutoReplyRepo) Save(row *model.CsAutoReplyRule) error {
	return r.db.Save(row).Error
}

func (r *AutoReplyRepo) Delete(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.CsAutoReplyRule{}, id).Error
}

func (r *AutoReplyRepo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&model.CsAutoReplyRule{}).Scopes(scopeTenant(r.tenantID)).Count(&n).Error
	return n, err
}
