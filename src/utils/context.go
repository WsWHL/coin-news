package utils

import (
	"github.com/gin-gonic/gin"
)

// response 响应结构
type response struct {
	Success bool           `json:"success"`
	Data    interface{}    `json:"data"`
	Error   *responseError `json:"error"`
	Message string         `json:"message"`
}

// responseError 错误响应
type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// pager 分页结构
type pager struct {
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	List  interface{} `json:"list"`
}

type ApiContext struct {
	*gin.Context
}

func ApiHandle(h func(*ApiContext)) gin.HandlerFunc {
	return func(c *gin.Context) {
		h(&ApiContext{c})
	}
}

// Ok 成功响应
func (c *ApiContext) Ok(data interface{}) {
	c.JSON(200, response{
		Success: true,
		Data:    data,
		Message: "请求成功",
	})
}

// Pager 分页响应
func (c *ApiContext) Pager(total, page, size int, list interface{}) {
	c.JSON(200, response{
		Success: true,
		Data: pager{
			Total: total,
			Page:  page,
			Size:  size,
			List:  list,
		},
		Message: "请求成功",
	})
}

// Error 错误响应
func (c *ApiContext) Error(code int, message string) {
	c.JSON(200, response{
		Success: false,
		Error: &responseError{
			Code:    code,
			Message: message,
		},
		Message: "发生错误",
	})
}
