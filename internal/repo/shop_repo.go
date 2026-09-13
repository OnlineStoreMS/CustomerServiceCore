package repo

import (
	"strings"
	"time"

	"customerservicecore/internal/model"

	"gorm.io/gorm"
)

type ShopRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewShopRepo(db *gorm.DB) *ShopRepo {
	return &ShopRepo{db: db}
}

func (r *ShopRepo) ForTenant(tenantID uint64) *ShopRepo {
	return &ShopRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *ShopRepo) List() ([]model.CsShop, error) {
	var list []model.CsShop
	err := r.db.Scopes(scopeTenant(r.tenantID)).Order("id DESC").Find(&list).Error
	return list, err
}

func (r *ShopRepo) Get(id uint64) (*model.CsShop, error) {
	var shop model.CsShop
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&shop, id).Error
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

func (r *ShopRepo) GetByBindCode(code string) (*model.CsShop, error) {
	var shop model.CsShop
	err := r.db.Where("bind_code = ?", strings.TrimSpace(code)).First(&shop).Error
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

func (r *ShopRepo) GetByPluginKey(key string) (*model.CsShop, error) {
	var shop model.CsShop
	err := r.db.Where("plugin_key = ?", strings.TrimSpace(key)).First(&shop).Error
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

func (r *ShopRepo) Create(shop *model.CsShop) error {
	shop.TenantID = r.tenantID
	return r.db.Create(shop).Error
}

func (r *ShopRepo) Save(shop *model.CsShop) error {
	return r.db.Save(shop).Error
}

func (r *ShopRepo) TouchHeartbeat(shop *model.CsShop) error {
	if shop == nil || shop.ID == 0 {
		return gorm.ErrRecordNotFound
	}
	return r.db.Model(&model.CsShop{}).Where("id = ?", shop.ID).Updates(map[string]any{
		"last_seen_at":  shop.LastSeenAt,
		"plugin_status": shop.PluginStatus,
		"updated_at":    time.Now(),
	}).Error
}
