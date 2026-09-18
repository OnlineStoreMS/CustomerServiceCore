package admin

import (
	"net/http"
	"strconv"

	"customerservicecore/internal/dto"
	"customerservicecore/internal/pkg/authcontext"
	"customerservicecore/internal/pkg/httputil"
	"customerservicecore/internal/pkg/response"
	"customerservicecore/internal/service"

	"github.com/gin-gonic/gin"
)

type ShopHandler struct {
	svc *service.ShopService
}

func NewShopHandler(svc *service.ShopService) *ShopHandler {
	return &ShopHandler{svc: svc}
}

func (h *ShopHandler) ss(c *gin.Context) *service.ShopService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *ShopHandler) List(c *gin.Context) {
	list, err := h.ss(c).List()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *ShopHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ShopHandler) Create(c *gin.Context) {
	var in dto.ShopCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *ShopHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.ShopUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ShopHandler) RotateBindCode(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).RotateBindCode(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ShopHandler) ResetPlugin(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).ResetPlugin(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

type ConversationHandler struct {
	svc *service.ShopService
}

func NewConversationHandler(svc *service.ShopService) *ConversationHandler {
	return &ConversationHandler{svc: svc}
}

func (h *ConversationHandler) ss(c *gin.Context) *service.ShopService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *ConversationHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	var shopID uint64
	if raw := c.Query("shopId"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid shopId")
			return
		}
		shopID = id
	}
	list, total, err := h.ss(c).ListConversations(shopID, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ConversationHandler) ClearAll(c *gin.Context) {
	if err := h.ss(c).ClearAllConversations(); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *ConversationHandler) Messages(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).ListMessages(id, page, pageSize)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ConversationHandler) Reply(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.ConversationReplyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).ReplyConversation(id, in.Content)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

type AutoReplyHandler struct {
	svc *service.ShopService
}

func NewAutoReplyHandler(svc *service.ShopService) *AutoReplyHandler {
	return &AutoReplyHandler{svc: svc}
}

func (h *AutoReplyHandler) ss(c *gin.Context) *service.ShopService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *AutoReplyHandler) List(c *gin.Context) {
	list, err := h.ss(c).ListAutoReplyRules()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *AutoReplyHandler) Create(c *gin.Context) {
	var in dto.AutoReplyRuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).CreateAutoReplyRule(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *AutoReplyHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.AutoReplyRuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).UpdateAutoReplyRule(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *AutoReplyHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).DeleteAutoReplyRule(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *AutoReplyHandler) GetLlm(c *gin.Context) {
	item, err := h.ss(c).GetLlmSetting()
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *AutoReplyHandler) SaveLlm(c *gin.Context) {
	var in dto.LlmSettingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).SaveLlmSetting(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *AutoReplyHandler) Seed(c *gin.Context) {
	list, err := h.ss(c).SeedAutoReplyPresets()
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, list)
}
