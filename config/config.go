package config

// 服务器相关配置
var (
	ListenPort = 8002
	ProjName   = "ezlock"
)

// 小程序相关配置
var (
	AppID  = "xxx"               // 微信小程序的appid
	Secret = "xxx" // 微信小程序的 secret
)

// jwt 相关的配置
var (
	JwtSecretKey = "xxxx" // jwt 密钥
)

// mongodb 相关配置
var (
	MgoUrl           = "mongodb://127.0.0.1:27017/admin"
	MgoTimeout       = 30
	MgoConnPoolLimit = 50
	DataBaseName     = "ezlcok"
)

// 门锁操作指令
var (
	OpenLock = "open"
	GetLog   = "getlog"
	AddCard  = "addcard"
	DelCard  = "delcard:%s"
)

func init() {

}
