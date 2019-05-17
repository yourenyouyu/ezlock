package utils

// 响应的错误码定义
const (
	OK = 0

	PARAM_ERR = 10000

	MONGO_ERR = 20000

	WEAPP_ERR = 30000

	UNAUTH = 40000

	NOT_EXISTS = 40001

	INVALID = 40002

	ENCRYPT_ERR = 50000
	DNCRYPT_ERR = 50001
)

// 错误码对应说明
var ERR_MSG_MAP = map[int]string{
	OK:          "OK",
	PARAM_ERR:   "参数错误",
	MONGO_ERR:   "数据库错误",
	WEAPP_ERR:   "微信小程序响应错误",
	UNAUTH:      "没有访问权限",
	NOT_EXISTS:  "不存在",
	INVALID:     "已失效",
	ENCRYPT_ERR: "加密数据失败",
	DNCRYPT_ERR: "解密数据失败",
}
