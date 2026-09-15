package repo

import (
	"strings"

	"customerservicecore/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConversationRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewConversationRepo(db *gorm.DB) *ConversationRepo {
	return &ConversationRepo{db: db}
}

func (r *ConversationRepo) ForTenant(tenantID uint64) *ConversationRepo {
	return &ConversationRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *ConversationRepo) List(shopID uint64, page, pageSize int) ([]model.CsConversation, int64, error) {
	q := r.db.Model(&model.CsConversation{}).Scopes(scopeTenant(r.tenantID))
	if shopID > 0 {
		q = q.Where("shop_id = ?", shopID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.CsConversation
	err := q.Order("last_message_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *ConversationRepo) Get(id uint64) (*model.CsConversation, error) {
	var conv model.CsConversation
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&conv, id).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepo) UpsertByBuyer(conv *model.CsConversation) error {
	conv.TenantID = r.tenantID
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "platform"},
			{Name: "platform_shop_id"},
			{Name: "platform_buyer_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"shop_id", "buyer_name", "platform_conversation_id",
			"last_message_at", "last_message_preview", "updated_at",
		}),
	}).Create(conv).Error
}

func (r *ConversationRepo) GetByBuyer(platform, platformShopID, platformBuyerID string) (*model.CsConversation, error) {
	var conv model.CsConversation
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("platform = ? AND platform_shop_id = ? AND platform_buyer_id = ?",
			strings.TrimSpace(platform), strings.TrimSpace(platformShopID), strings.TrimSpace(platformBuyerID)).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepo) Save(conv *model.CsConversation) error {
	return r.db.Save(conv).Error
}

func (r *ConversationRepo) ListByShop(shopID uint64) ([]model.CsConversation, error) {
	q := r.db.Scopes(scopeTenant(r.tenantID))
	if shopID > 0 {
		q = q.Where("shop_id = ?", shopID)
	}
	var list []model.CsConversation
	err := q.Order("id ASC").Find(&list).Error
	return list, err
}

func (r *ConversationRepo) Delete(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.CsConversation{}, id).Error
}
