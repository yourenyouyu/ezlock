package mongo

import (
	"ezlock/config"
	"fmt"
	"github.com/globalsign/mgo"
	"os"
	"time"
)

var mgoSession *mgo.Session

func init() {
	// 链接到mongo 获取操作 mongo 的session
	var err error
	mgoSession, err = mgo.DialWithTimeout(config.MgoUrl, time.Duration(config.MgoTimeout)*time.Second)
	if err != nil {
		fmt.Printf("mgo connect occur error [%s]", err.Error())
		os.Exit(1)
	}
	// 设置 如果主可用的可以的话 优先从主服务读，保证数据比较新
	mgoSession.SetMode(mgo.PrimaryPreferred, true)
	// 设置连接池的最大值，默认4096
	mgoSession.SetPoolLimit(config.MgoConnPoolLimit)
	// 设置每一个mongo操作的超时时间，此处用默认值7秒
	//mgoSession.SetSyncTimeout()
}

// 从连接池中获取一个session
func GetMgoSession() *mgo.Session {
	return mgoSession.Copy()
}

// 连接使用完毕后关闭连接
func PutMgoSession(session *mgo.Session) {
	if session != nil {
		session.Close()
	}
}
