package repo

import (
	"customerservicecore/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MessageRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewMessageRepo(db *gorm.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) ForTenant(tenantID uint64) *MessageRepo {
	return &MessageRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *MessageRepo) ListByConversation(conversationID uint64, page, pageSize int) ([]model.CsMessage, int64, error) {
	q := r.db.Model(&model.CsMessage{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("conversation_id = ?", conversationID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.CsMessage
	err := q.Order("sent_at ASC, id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *MessageRepo) CreateIgnoreDuplicate(msg *model.CsMessage) (bool, error) {
	msg.TenantID = r.tenantID
	tx := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "platform"}, {Name: "platform_message_id"}},
		DoNothing: true,
	}).Create(msg)
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}
