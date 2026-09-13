package repo

import "gorm.io/gorm"

type Repos struct {
	Shop         *ShopRepo
	Conversation *ConversationRepo
	Message      *MessageRepo
}

func New(db *gorm.DB) *Repos {
	return &Repos{
		Shop:         NewShopRepo(db),
		Conversation: NewConversationRepo(db),
		Message:      NewMessageRepo(db),
	}
}

func NormalizeTenantID(id uint64) uint64 {
	if id == 0 {
		return 1
	}
	return id
}

func scopeTenant(tenantID uint64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ?", NormalizeTenantID(tenantID))
	}
}
