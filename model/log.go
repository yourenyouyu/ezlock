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
var LogTableName = "Log"

// 表结构
type Log struct {
	// omitempty如果不是空值才包含_id,是空值就不包含，这样的mongo可以自动生成，不写omitempty，每次插入的时候就必须要传_id了
	Id         bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	LockId     bson.ObjectId `json:"lockId" bson:"lockId"`         // 门锁id
	UserId     bson.ObjectId `json:"userId" bson:"userId"`         // 开锁用户
	OpenType   string        `json:"openType" bson:"openType"`     // 开锁类型
	Success    bool          `json:"success" bson:"success"`       // 开锁是否成功
	RowInfo    string        `json:"rawInfo" bson:"rawInfo"`       // 硬件存储的原始信息
	CreateTime time.Time     `json:"createTime" bson:"createTime"` // 写入时间
}

// 创建表的时候初始化一些操作，比如建立索引
func init() {
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)
	// 连接到当前表
	coll := mgoSession.DB(config.DataBaseName).C(LockTableName)
	err := coll.EnsureIndex(mgo.Index{
		Key:  []string{"openType"},
		Name: "Index_OpenType",
	})

	if err != nil {
		fmt.Printf("Log Create Index_OpenType Failed: %s\n", err.Error())
		os.Exit(1)
	}
}
