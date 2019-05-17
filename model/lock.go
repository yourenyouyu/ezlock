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
var LockTableName = "Lock"

// 表结构
type Lock struct {
	// omitempty如果不是空值才包含_id,是空值就不包含，这样的mongo可以自动生成，不写omitempty，每次插入的时候就必须要传_id了
	Id         bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	Name       string        `json:"name" bson:"name"`             // 锁名称
	Mac        string        `json:"mac" bson:"mac"`               // mac 地址
	Desc       string        `json:"desc" bson:"desc"`             // 锁的描述信息
	Model      string        `json:"model" bson:"model"`           // 硬件型号
	Version    string        `json:"version" bson:"version"`       // 软件版本
	Key        string        `json:"key" bson:"key"`               // 加密密钥
	Own        bson.ObjectId `json:"own" bson:"own,omitempty"`     // 门锁拥有者，就是购买者
	Valid      bool          `json:"valid" bson:"valid"`           // 门锁是否有效
	UpdateTime time.Time     `json:"updateTime" bson:"updateTime"` // 更新时间
	CreateTime time.Time     `json:"createTime" bson:"createTime"` // 写入时间
}

// 创建表的时候初始化一些操作，比如建立索引
func init() {
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)
	// 连接到当前表
	coll := mgoSession.DB(config.DataBaseName).C(LockTableName)
	// 建立name索引, 方便查询
	err := coll.EnsureIndex(mgo.Index{
		Key:  []string{"name"},
		Name: "Index_Name",
	})

	if err != nil {
		fmt.Printf("Lock Create Index_Name Failed: %s\n", err.Error())
		os.Exit(1)
	}

	err = coll.EnsureIndex(mgo.Index{
		Key:    []string{"mac"},
		Unique: true,
		Name:   "Index_Mac",
	})

	if err != nil {
		fmt.Printf("Lock Create Index_Mac Failed: %s\n", err.Error())
		os.Exit(1)
	}

	// 建立创建时间的 倒叙 索引
	err = coll.EnsureIndex(mgo.Index{
		Key:  []string{"-createTime"},
		Name: "Index_CreateTime",
	})

	if err != nil {
		fmt.Printf("Lock Create Index_CreateTime Failed: %s\n", err.Error())
		os.Exit(1)
	}
}
