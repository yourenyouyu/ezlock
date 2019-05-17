package model

import (
	"ezlock/common/mongo"
	"ezlock/config"
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"os"
	"time"
)

// 门锁信息表名称
var UserTableName = "User"

// 表结构
type User struct {
	// omitempty如果不是空值才包含_id,是空值就不包含，这样的mongo可以自动生成，不写omitempty，每次插入的时候就必须要传_id了
	Id          bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	NickName    string        `json:"nickName" bson:"nickName"`                 // 用户昵称
	OpenId      string        `json:"openId" bson:"openId"`                     // 微信的 openId
	UnionId     string        `json:"unionId" bson:"unionId"`                   // 微信的 openId
	SessionKey  string        `json:"sessionKey" bson:"sessionKey"`             // 微信 服务器返回的session key
	PhoneNumber string        `json:"phoneNumber" bson:"phoneNumber"`           // 用户手机号码
	DefaultLock bson.ObjectId `json:"defaultLock" bson:"defaultLock,omitempty"` // 用户默认拥有的锁
	Gender      int           `json:"gender" bson:"gender"`                     // 用户性别
	City        string        `json:"city" bson:"city"`                         // 用户所在城市
	Province    string        `json:"province" bson:"province"`                 // 用户所在省份
	Country     string        `json:"country" bson:"country"`                   // 用户所在国家
	AvatarUrl   string        `json:"avatarUrl" bson:"avatarUrl"`               // 用户头像链接
	Language    string        `json:"language" bson:"language"`
	UpdateTime  time.Time     `json:"updateTime" bson:"updateTime"` // 更新时间
	CreateTime  time.Time     `json:"createTime" bson:"createTime"` // 写入时间
}

// 创建表的时候初始化一些操作，比如建立索引
func init() {
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)
	// 连接到当前表
	coll := mgoSession.DB(config.DataBaseName).C(UserTableName)
	// 建立name索引, 方便查询
	err := coll.EnsureIndex(mgo.Index{
		Key:  []string{"nickName"},
		Name: "Index_NickName",
	})

	if err != nil {
		fmt.Printf("User Create Index_NickName Failed: %s\n", err.Error())
		os.Exit(1)
	}

	err = coll.EnsureIndex(mgo.Index{
		Key:    []string{"openId"},
		Unique: true,
		Name:   "Index_OpenId",
	})

	if err != nil {
		fmt.Printf("User Create Index_OpenId Failed: %s\n", err.Error())
		os.Exit(1)
	}

	// 建立创建时间的 倒叙 索引
	err = coll.EnsureIndex(mgo.Index{
		Key:  []string{"-createTime"},
		Name: "Index_CreateTime",
	})

	if err != nil {
		fmt.Printf("User Create Index_CreateTime Failed: %s\n", err.Error())
		os.Exit(1)
	}
}
