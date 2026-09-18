package plugin

import (
	"net/http"
	"strings"

	"customerservicecore/internal/dto"
	"customerservicecore/internal/model"
	"customerservicecore/internal/pkg/httputil"
	"customerservicecore/internal/pkg/response"
	"customerservicecore/internal/service"

	"github.com/gin-gonic/gin"
)

const contextShop = "plugin_shop"

type Handler struct {
	svc *service.ShopService
}

func NewHandler(svc *service.ShopService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Bind(c *gin.Context) {
	var in dto.PluginBindInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Bind(in.BindCode)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		key, secret := pluginCreds(c)
		shop, err := h.svc.AuthenticatePlugin(key, secret)
		if err != nil {
			httputil.HandleServiceError(c, err)
			c.Abort()
			return
		}
		c.Set(contextShop, shop)
		c.Next()
	}
}

func (h *Handler) Heartbeat(c *gin.Context) {
	shop := mustShop(c)
	if shop == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrPluginAuth.Error())
		return
	}
	item, err := h.svc.Heartbeat(shop)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) Messages(c *gin.Context) {
	shop := mustShop(c)
	if shop == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrPluginAuth.Error())
		return
	}
	var in dto.PluginMessagesInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.IngestMessages(shop, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) ClaimOutbound(c *gin.Context) {
	shop := mustShop(c)
	if shop == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrPluginAuth.Error())
		return
	}
	list, err := h.svc.ClaimOutbound(shop, 5)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Handler) AckOutbound(c *gin.Context) {
	shop := mustShop(c)
	if shop == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrPluginAuth.Error())
		return
	}
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.PluginOutboundAckInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.AckOutbound(shop, id, in.OK, in.Error); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *Handler) Unbind(c *gin.Context) {
	shop := mustShop(c)
	if shop == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrPluginAuth.Error())
		return
	}
	if err := h.svc.UnbindPlugin(shop); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *Handler) SetMonitor(c *gin.Context) {
	shop := mustShop(c)
	if shop == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrPluginAuth.Error())
		return
	}
	var in dto.PluginMonitorInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.SetPluginMonitor(shop, in.Enabled)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func pluginCreds(c *gin.Context) (key, secret string) {
	key = strings.TrimSpace(c.GetHeader("X-Plugin-Key"))
	secret = strings.TrimSpace(c.GetHeader("X-Plugin-Secret"))
	return key, secret
}

func mustShop(c *gin.Context) *model.CsShop {
	v, _ := c.Get(contextShop)
	shop, _ := v.(*model.CsShop)
	return shop
}
