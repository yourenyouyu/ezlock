package utils

import (
	"ezlock/common/mongo"
	"ezlock/config"
	"ezlock/model"
	"github.com/globalsign/mgo/bson"
	"time"
)

// 检测对应授权类型是否有效
func CheckAuthValid(auth model.Auth) bool {
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)
	q := bson.M{
		"_id":   auth.LockId,
		"valid": true,
	}
	lock := model.Lock{}
	// 查看门锁是否被删除
	err := lockColl.Find(q).Select(bson.M{"_id": 1}).One(&lock)
	if err != nil {
		// err 可能是没发现，也可能是其他数据库错误，此处直接设置为无效授权
		return false
	}
	return CheckAuthTimeValid(auth)
}

func CheckAuthTimeValid(auth model.Auth) bool {
	switch auth.AuthType {
	case "1":
		return true
	case "2":
		// 一次性授权 2019-10-1 0:00
		deadLine, err := time.ParseInLocation("2006-01-02 15:04", auth.Deadline, time.Local)
		if err != nil {
			return false
		}
		now := time.Now().Local()
		if now.After(deadLine) {
			return false
		}
		return true
	case "3":
		// 时间段授权
		// 一次性授权 2019-10-1 0:00
		startDate, err := time.ParseInLocation("2006-01-02", auth.StartDate, time.Local)
		if err != nil {
			return false
		}
		endDate, err := time.ParseInLocation("2006-01-02", auth.EndDate, time.Local)
		if err != nil {
			return false
		}
		startTime, err := time.ParseInLocation("15:04", auth.StartTime, time.Local)
		if err != nil {
			return false
		}
		endTime, err := time.ParseInLocation("15:04", auth.EndTime, time.Local)
		if err != nil {
			return false
		}

		now := time.Now().Local()
		// 先判断是否在制定日期，不在的话 就响应过期
		if now.Before(startDate) || now.After(endDate) {
			return false
		}
		currentTime, err := time.ParseInLocation("15:04", now.Format("15:04"), time.Local)
		if err != nil {
			return false
		}
		// 在判断是否在制定时段，不在的话 就响应过期
		if currentTime.Before(startTime) || currentTime.After(endTime) {
			return false
		}
		return true

	}
	return false
}
