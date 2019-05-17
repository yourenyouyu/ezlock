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
var CardTableName = "Card"

// 表结构
type Card struct {
	// omitempty如果不是空值才包含_id,是空值就不包含，这样的mongo可以自动生成，不写omitempty，每次插入的时候就必须要传_id了
	Id         bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	Name       string        `json:"name" bson:"name"`             // 锁名称
	Number     string        `json:"number" bson:"number"`         // 门禁卡号码
	Desc       string        `json:"desc" bson:"desc"`             // 锁的描述信息
	Lock       bson.ObjectId `json:"lock" bson:"lock"`             // 门禁卡绑定的锁
	UserId     bson.ObjectId `json:"userId" bson:"userId"`         // 门卡的添加者
	Valid      bool          `json:"valid" bson:"valid"`           // 门卡是否有效
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
		Key:    []string{"lock"},
		Unique: true,
		Name:   "Index_Lock",
	})

	if err != nil {
		fmt.Printf("Lock Create Index_Lock Failed: %s\n", err.Error())
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
