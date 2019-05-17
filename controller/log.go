package controller

import (
	"ezlock/common/mongo"
	"ezlock/config"
	"ezlock/model"
	"ezlock/utils"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"strings"
	"time"
)

func GetLogKey(c *gin.Context) {
	userId := c.GetString("id")
	// 请求参数列表
	params := &struct {
		Code string `form:"code" binding:"len=16,required"`
		Mac  string `form:"mac" binding:"required"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}
	key, err := utils.GenerateKey(userId, params.Mac, config.GetLog, params.Code)
	if err != nil {
		utils.ResponseError(utils.ENCRYPT_ERR, err.Error(), c)
		return
	}

	utils.ResponseOk(key, c)
}

type LogDetail struct {
	model.Log
	User   string        `json:"user"`
	LockId bson.ObjectId `json:"-"` // 被授权的门锁id
}

func GetLockOperateLog(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		Mac string `form:"mac"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}
	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)
	logColl := mgoSession.DB(config.DataBaseName).C(model.LogTableName)
	userColl := mgoSession.DB(config.DataBaseName).C(model.UserTableName)

	lock := model.Lock{}
	err := lockColl.Find(bson.M{
		"mac": params.Mac,
	}).Select(bson.M{"_id": 1}).One(&lock)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	locks, err := utils.GetAllLocks(userId, false, model.Perms{
		ViewLog: true,
	})
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	// 要查看的门锁，不在用户所有的锁的列表中，则不让查看日志
	if _, ok := locks[lock.Id]; !ok {
		utils.ResponseError(utils.NOT_EXISTS, "您无权查看此锁的开锁日志", c)
		return
	}

	q := bson.M{
		"lockId": lock.Id,
	}
	logs := []LogDetail{}
	err = logColl.Find(q).All(&logs)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	for index := range logs {
		user := model.User{}
		err = userColl.FindId(logs[index].UserId).Select(bson.M{"nickName": 1}).One(&user)
		if err != nil {
			utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
			return
		}

		logs[index].User = user.NickName
	}
	utils.ResponseOk(logs, c)
}

// 硬件需要对锁的日志信息做个加密防止篡改，前端小程序蓝牙链接成功后 拿到这个加密信息直接发送给后端
func SetLockOperateLog(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		Mac  string `form:"mac" binding:"required"`
		Data string `form:"data" binding:"required"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}
	content, err := utils.DncryptData(userId, params.Mac, params.Data)
	if err != nil {
		utils.ResponseError(utils.DNCRYPT_ERR, err.Error(), c)
		return
	}

	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	logColl := mgoSession.DB(config.DataBaseName).C(model.LogTableName)
	userColl := mgoSession.DB(config.DataBaseName).C(model.UserTableName)
	lockColl := mgoSession.DB(config.DataBaseName).C(model.LogTableName)

	lock := model.Lock{}
	err = lockColl.Find(bson.M{"mac": params.Mac}).Select(bson.M{"_id": 1}).One(&lock)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	// 操作指令_时间_方式_卡号/操作用户_1,操作指令_锁的mac地址_方式_卡号/操作用户
	opLogs := strings.Split(strings.TrimSpace(content), ",")
	for _, opLog := range opLogs {
		log := strings.Split(strings.TrimSpace(opLog), "_")
		if len(log) != 5 {
			// 日志格式错误
			continue
		}
		opTime, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(log[1]), time.Local)
		if err != nil {
			// 日志格式错误
			continue
		}

		method := strings.TrimSpace(log[2])
		info := strings.TrimSpace(log[3])
		success := false
		if strings.TrimSpace(log[4]) == "1" {
			success = true
		}
		logInfo := model.Log{
			LockId:     lock.Id,
			CreateTime: opTime,
			Success:    success,
			OpenType:   method,
			RowInfo:    strings.TrimSpace(opLog),
		}
		switch method {
		case "0":
			// 门卡开锁
		case "1":
			// 蓝牙开锁
			userId := info
			user := model.User{}
			err := userColl.FindId(userId).One(&user)
			if err != nil {
				// 日志格式错误
				continue
			}
			logInfo.UserId = user.Id

		}
		_, err = logColl.Upsert(bson.M{"rawInfo": strings.TrimSpace(opLog)}, logInfo)
		if err != nil {
			// 日志格式错误
			continue
		}
	}

	utils.ResponseOk("ok", c)
}
