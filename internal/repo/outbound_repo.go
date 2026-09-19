package repo

import (
	"time"

	"customerservicecore/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OutboundRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewOutboundRepo(db *gorm.DB) *OutboundRepo {
	return &OutboundRepo{db: db}
}

func (r *OutboundRepo) ForTenant(tenantID uint64) *OutboundRepo {
	return &OutboundRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *OutboundRepo) Create(row *model.CsOutboundMessage) error {
	row.TenantID = r.tenantID
	return r.db.Create(row).Error
}

func (r *OutboundRepo) Get(id uint64) (*model.CsOutboundMessage, error) {
	var row model.CsOutboundMessage
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *OutboundRepo) Save(row *model.CsOutboundMessage) error {
	return r.db.Save(row).Error
}

func (r *OutboundRepo) CountPending(shopID uint64) (int64, error) {
	var n int64
	q := r.db.Model(&model.CsOutboundMessage{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status IN ?", []string{model.OutboundPending, model.OutboundSending})
	if shopID > 0 {
		q = q.Where("shop_id = ?", shopID)
	}
	err := q.Count(&n).Error
	return n, err
}

func (r *OutboundRepo) HasRecentAuto(conversationID uint64, since time.Time) (bool, error) {
	return r.HasRecentSource(conversationID, since, model.ReplySourceAuto, model.ReplySourceLlm)
}

func (r *OutboundRepo) HasRecentSource(conversationID uint64, since time.Time, sources ...string) (bool, error) {
	if len(sources) == 0 {
		return r.HasRecentAuto(conversationID, since)
	}
	var n int64
	err := r.db.Model(&model.CsOutboundMessage{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("conversation_id = ? AND source IN ? AND created_at >= ? AND status <> ?",
			conversationID, sources, since, model.OutboundFailed).
		Count(&n).Error
	return n > 0, err
}

func (r *OutboundRepo) ExistsTrigger(triggerID string) (bool, error) {
	if triggerID == "" {
		return false, nil
	}
	var n int64
	err := r.db.Model(&model.CsOutboundMessage{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("trigger_platform_msg_id = ? AND status <> ?", triggerID, model.OutboundFailed).
		Count(&n).Error
	return n > 0, err
}

func (r *OutboundRepo) RecoverStale(shopID uint64, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	q := r.db.Model(&model.CsOutboundMessage{}).
		Scopes(scopeTenant(r.tenantID)).
		Where("status = ? AND updated_at < ?", model.OutboundSending, cutoff)
	if shopID > 0 {
		q = q.Where("shop_id = ?", shopID)
	}
	return q.Update("status", model.OutboundPending).Error
}

func (r *OutboundRepo) DeleteByShop(shopID uint64) error {
	q := r.db.Scopes(scopeTenant(r.tenantID)).Where("id > ?", 0)
	if shopID > 0 {
		q = q.Where("shop_id = ?", shopID)
	}
	return q.Delete(&model.CsOutboundMessage{}).Error
}

func (r *OutboundRepo) ClaimPending(shopID uint64, limit int) ([]model.CsOutboundMessage, error) {
	if limit <= 0 {
		limit = 3
	}
	_ = r.RecoverStale(shopID, 2*time.Minute)
	var list []model.CsOutboundMessage
	err := r.db.Transaction(func(tx *gorm.DB) error {
		q := tx.Where("tenant_id = ? AND status = ?", r.tenantID, model.OutboundPending)
		if tx.Dialector.Name() == "postgres" {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if shopID > 0 {
			q = q.Where("shop_id = ?", shopID)
		}
		if err := q.Order("id ASC").Limit(limit).Find(&list).Error; err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		ids := make([]uint64, 0, len(list))
		for i := range list {
			ids = append(ids, list[i].ID)
			list[i].Status = model.OutboundSending
		}
		return tx.Model(&model.CsOutboundMessage{}).
			Where("id IN ?", ids).
			Update("status", model.OutboundSending).Error
	})
	return list, err
}
