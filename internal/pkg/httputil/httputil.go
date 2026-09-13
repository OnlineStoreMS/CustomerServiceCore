package httputil

import (
	"errors"
	"net/http"
	"strconv"

	"customerservicecore/internal/pkg/response"
	"customerservicecore/internal/service"

	"github.com/gin-gonic/gin"
)

func ParseID(c *gin.Context) (uint64, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func ParsePage(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	return page, pageSize
}

func HandleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrPluginAuth):
		response.Fail(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrBindCodeInvalid), errors.Is(err, service.ErrAlreadyBound):
		response.Fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrBadRequest):
		response.Fail(c, http.StatusBadRequest, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, err.Error())
	}
}
