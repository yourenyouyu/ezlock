package controller

import (
	"ezlock/common/mongo"
	"ezlock/config"
	"ezlock/model"
	"ezlock/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"time"
)

type AuthDetail struct {
	model.Auth
	Sender   string        `json:"sender"`   // 发送者
	Receiver string        `json:"receiver"` // 接受者
	LockId   bson.ObjectId `json:"-"`        // 被授权的门锁id
}

func GetLockAuthList(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		Mac string `form:"mac" binding:"required"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}
	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)
	authColl := mgoSession.DB(config.DataBaseName).C(model.AuthTableName)
	userColl := mgoSession.DB(config.DataBaseName).C(model.UserTableName)
	authLock := model.Lock{}
	err := lockColl.Find(bson.M{
		"mac": params.Mac,
	}).Select(bson.M{"_id": 1}).One(&authLock)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	q := bson.M{
		"lockId": authLock.Id,
		"$or": []bson.M{
			bson.M{"sendId": bson.ObjectIdHex(userId)},
			bson.M{"receiverId": bson.ObjectIdHex(userId)},
		},
	}
	auths := []AuthDetail{}
	err = authColl.Find(q).All(&auths)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	for index := range auths {
		sender := model.User{}
		receiver := model.User{}
		err = userColl.FindId(auths[index].SendId).Select(bson.M{"nickName": 1}).One(&sender)
		if err != nil {
			utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
			return
		}

		err = userColl.FindId(auths[index].ReceiverId).Select(bson.M{"nickName": 1}).One(&receiver)
		if err != nil {
			utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
			return
		}

		auths[index].Sender = sender.NickName
		auths[index].Receiver = receiver.NickName
	}
	utils.ResponseOk(auths, c)
}

func CreateLockAuth(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		Mac       string `form:"mac" binding:"required"`
		ViewLog   bool   `form:"viewLog"`   // 查看日志权限
		AddCard   bool   `form:"addCard"`   // 添加门卡权限
		ShareAuth bool   `form:"shareAuth"` // 分享授权权限
		AuthType  string `form:"authType" binding:"required"`
		Deadline  string `form:"deadline"`
		StartDate string `form:"startDate"`
		EndDate   string `form:"endDate"`
		StartTime string `form:"startTime"`
		EndTime   string `form:"endTime"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}
	authInfo := model.Auth{
		SendId:     bson.ObjectIdHex(userId),
		AuthType:   params.AuthType,
		Deadline:   params.Deadline,
		StartDate:  params.StartDate,
		EndDate:    params.EndDate,
		StartTime:  params.StartTime,
		EndTime:    params.EndTime,
		Valid:      true,
		UpdateTime: time.Now().Local(),
		CreateTime: time.Now().Local(),
	}
	authInfo.Perms = model.Perms{
		ShareAuth: params.ShareAuth,
		AddCard:   params.AddCard,
		ViewLog:   params.ViewLog,
	}
	if !utils.CheckAuthTimeValid(authInfo) {
		utils.ResponseError(utils.PARAM_ERR, "授权时间不合法", c)
		return
	}

	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)
	authColl := mgoSession.DB(config.DataBaseName).C(model.AuthTableName)
	q := bson.M{
		"mac": params.Mac,
	}
	lock := model.Lock{}
	err := lockColl.Find(q).Select(bson.M{"_id": 1}).One(&lock)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	// 判断默认锁是否在可用锁列表里面，不在 则默认锁已失效
	locks, err := utils.GetAllLocks(userId, true, model.Perms{
		ShareAuth: true,
	})
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	isOwn, ok := locks[lock.Id]
	if !ok {
		utils.ResponseError(utils.UNAUTH, "您无权分享此锁的授权", c)
		return
	}
	// 如果这个锁不是自己的，没有办法让别人在分享授权
	if !isOwn && authInfo.ShareAuth {
		utils.ResponseError(utils.UNAUTH, "您无权给别人开放此锁的分享权限", c)
		return
	}

	mgoId := bson.NewObjectId()
	authInfo.LockId = lock.Id
	authInfo.Id = mgoId
	authInfo.Token = mgoId.Hex()
	err = authColl.Insert(authInfo)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	utils.ResponseOk(authInfo.Token, c)

}

func UseLockAuth(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		Token string `form:"token" binding:"required"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}

	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	authColl := mgoSession.DB(config.DataBaseName).C(model.AuthTableName)
	q := bson.M{
		"_id": bson.ObjectIdHex(params.Token),
	}
	// 后面得加分布式锁，高并发这么写会有问题
	authInfo := model.Auth{}
	err := authColl.Find(q).One(&authInfo)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	// 此授权已经被别人使用
	if len(authInfo.ReceiverId.Hex()) != 0 {
		utils.ResponseError(utils.INVALID, "已被他人使用", c)
		return
	}
	if authInfo.SendId.Hex() == userId {
		utils.ResponseError(utils.PARAM_ERR, "自己不能使用自己的授权哦！", c)
		return
	}

	err = authColl.Update(q, bson.M{"receiverId": bson.ObjectIdHex(userId)})
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	utils.ResponseOk("ok", c)

}

func RevokeAuth(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		AuthId string `form:"authId" binding:"required"`
	}{}

	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	authColl := mgoSession.DB(config.DataBaseName).C(model.AuthTableName)

	updateVal := &struct {
		Valid      bool      `bson:"valid"`
		UpdateTime time.Time `bson:"updateTime"` // 更新时间
	}{
		Valid:      false,
		UpdateTime: time.Now().Local(),
	}

	if err := authColl.Update(bson.M{
		"_id":    bson.ObjectIdHex(params.AuthId),
		"sendId": bson.ObjectIdHex(userId),
	}, bson.M{
		"$set": updateVal,
	}); err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	utils.ResponseOk(fmt.Sprintf("auth[%s] revoke success", params.AuthId), c)

}
