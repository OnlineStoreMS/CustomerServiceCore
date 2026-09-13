package service

import "errors"

var (
	ErrNotFound        = errors.New("记录不存在")
	ErrBadRequest      = errors.New("请求参数无效")
	ErrPluginAuth      = errors.New("插件鉴权失败")
	ErrAlreadyBound    = errors.New("店铺已绑定，请先重置插件")
	ErrBindCodeInvalid = errors.New("绑定码无效")
)
