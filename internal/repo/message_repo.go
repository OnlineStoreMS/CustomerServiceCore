package repo

import (
	"strings"

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

func (r *MessageRepo) ListRecentByConversation(conversationID uint64, limit int) ([]model.CsMessage, error) {
	if limit <= 0 {
		limit = 6
	}
	var list []model.CsMessage
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("conversation_id = ?", conversationID).
		Order("id DESC").
		Limit(limit).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, nil
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

func (r *MessageRepo) DeleteByShop(shopID uint64) error {
	if shopID == 0 {
		return nil
	}
	return r.db.Model(&model.CsMessage{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("shop_id = ?", shopID).
		Delete(&model.CsMessage{}).Error
}

func (r *MessageRepo) ReassignConversation(fromID, toID uint64) error {
	if fromID == 0 || toID == 0 || fromID == toID {
		return nil
	}
	return r.db.Model(&model.CsMessage{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("conversation_id = ?", fromID).
		Update("conversation_id", toID).Error
}

func (r *MessageRepo) ListByConversationIDs(ids []uint64) ([]model.CsMessage, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.CsMessage
	err := r.db.Model(&model.CsMessage{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("conversation_id IN ?", ids).
		Order("sent_at ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *MessageRepo) DeduplicateConversation(conversationID uint64) error {
	if conversationID == 0 {
		return nil
	}
	var list []model.CsMessage
	if err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("conversation_id = ?", conversationID).
		Order("sent_at ASC, id ASC").
		Find(&list).Error; err != nil {
		return err
	}
	seen := map[string]struct{}{}
	var dupes []uint64
	for i := range list {
		key := list[i].Direction + "\n" + strings.TrimSpace(list[i].Content)
		if _, ok := seen[key]; ok {
			dupes = append(dupes, list[i].ID)
			continue
		}
		seen[key] = struct{}{}
	}
	if len(dupes) == 0 {
		return nil
	}
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.CsMessage{}, dupes).Error
}
