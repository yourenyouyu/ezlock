package model

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

// 门锁信息表名称
var AuthTableName = "Auth"

type Perms struct {
	ViewLog   bool `json:"viewLog" bson:"viewLog"`     // 查看日志权限
	AddCard   bool `json:"addCard" bson:"addCard"`     // 添加门卡权限
	ShareAuth bool `json:"shareAuth" bson:"shareAuth"` // 分享授权权限
}

// 表结构
type Auth struct {
	Perms
	// omitempty如果不是空值才包含_id,是空值就不包含，这样的mongo可以自动生成，不写omitempty，每次插入的时候就必须要传_id了
	Id         bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	SendId     bson.ObjectId `json:"sendId" bson:"sendId"`                             // 发送者id
	ReceiverId bson.ObjectId `json:"receiverId,omitempty" bson:"receiverId,omitempty"` // 发送者id
	LockId     bson.ObjectId `json:"lockId" bson:"lockId"`                             // 被授权的门锁id
	AuthType   string        `json:"authType" bson:"authType"`                         // 授权类型
	Deadline   string        `json:"deadline" bson:"deadline"`                         // 截止时间
	StartDate  string        `json:"startDate" bson:"startDate"`                       // 授权开始日期
	EndDate    string        `json:"endDate" bson:"endDate"`                           // 授权结束日期
	StartTime  string        `json:"startTime" bson:"startTime"`                       // 授权开始时间
	EndTime    string        `json:"endTime" bson:"endTime"`                           // 授权结束时间
	Valid      bool          `json:"valid" bson:"valid"`                               // 授权是否有效
	Token      string        `json:"token" bson:"token"`                               // 一次性授权时携带的token
	UpdateTime time.Time     `json:"updateTime" bson:"updateTime"`                     // 更新时间
	CreateTime time.Time     `json:"createTime" bson:"createTime"`                     // 写入时间
}

// 创建表的时候初始化一些操作，比如建立索引
func init() {

}
