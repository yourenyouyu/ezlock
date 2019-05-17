package controller

import (
	"ezlock/common/mongo"
	"ezlock/config"
	"ezlock/model"
	"ezlock/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"time"
)

func GetDefaultLock(c *gin.Context) {
	userId := c.GetString("id")

	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)
	userColl := mgoSession.DB(config.DataBaseName).C(model.UserTableName)

	user := model.User{}
	err := userColl.FindId(userId).Select(bson.M{"_id": 0, "defaultLock": 1}).One(&user)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	// 用户没有默认锁，注意 加锁的时候判断用户有没有默认锁，没有以加入的第一把锁作为默认锁
	if len(user.DefaultLock.Hex()) == 0 {
		utils.ResponseError(utils.NOT_EXISTS, "您没有可用使用的门锁", c)
		return
	}
	defaultLockId := user.DefaultLock

	// 响应给用户的结构
	resp := &struct {
		Name  string `json:"name" bson:"name"`
		Mac   string `json:"mac" bson:"mac"`
		Valid bool   `json:"valid" bson:"valid"` // valid字段表示这个默认锁是否有效，比如用户将授权锁作为自己的默认锁，授权过期，则这个字段就是false
	}{}
	err = lockColl.FindId(defaultLockId).Select(bson.M{"_id": 0, "name": 1, "mac": 1}).One(resp)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	// 判断默认锁是否在可用锁列表里面，不在 则默认锁已失效
	locks, err := utils.GetAllLocks(userId, true, model.Perms{})
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	if _, ok := locks[defaultLockId]; ok {
		resp.Valid = true
	}
	utils.ResponseOk(resp, c)
}

func SetDefaultLock(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		Mac string `form:"mac" binding:"required"`
	}{}
	// 看设置的默认锁是否在用户所有的锁的列表中，即使锁无效也可用设置为默认锁 防止是时段授权
	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)
	userColl := mgoSession.DB(config.DataBaseName).C(model.UserTableName)
	q := bson.M{
		"mac": params.Mac,
	}
	defaultLock := model.Lock{}
	err := lockColl.Find(q).Select(bson.M{"_id": 1}).One(&defaultLock)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	defaultLockId := defaultLock.Id

	// 判断默认锁是否在可用锁列表里面，不在 则默认锁已失效
	locks, err := utils.GetAllLocks(userId, false, model.Perms{})
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	// 要设置的门锁，不在用户所有的锁的列表中，则不让设置
	if _, ok := locks[defaultLockId]; !ok {
		utils.ResponseError(utils.NOT_EXISTS, "没发现您要设置的门锁", c)
		return
	}

	// 设置默认锁
	updateVal := bson.M{
		"defaultLock": defaultLockId,
		"updateTime":  time.Now().Local(),
	}
	if err := userColl.UpdateId(userId, bson.M{
		"$set": updateVal,
	}); err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	utils.ResponseOk("ok", c)
}

func GetOpenLockKey(c *gin.Context) {
	userId := c.GetString("id")
	// 请求参数列表
	params := &struct {
		Code string `form:"code" binding:"len=16,required"`
		Mac  string `form:"mac" binding:"required"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}
	key, err := utils.GenerateKey(userId, params.Mac, config.OpenLock, params.Code)
	if err != nil {
		utils.ResponseError(utils.ENCRYPT_ERR, err.Error(), c)
		return
	}
	// 获取mongo的操作session

	utils.ResponseOk(key, c)
}

func GetLockList(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		ShowValid bool `form:"showValid"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}
	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)

	locks, err := utils.GetAllLocks(userId, params.ShowValid, model.Perms{})
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	lockIds := make([]bson.ObjectId, len(locks))
	for key := range locks {
		lockIds = append(lockIds, key)
	}

	q := bson.M{
		"_id": bson.M{"$in": lockIds},
	}
	resp := []model.Lock{}
	// 响应给用户的结构
	err = lockColl.Find(q).Select(bson.M{"key": 0, "own": 0}).All(&resp)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	utils.ResponseOk(resp, c)
}

func AddLock(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		Name    string `form:"name" binding:"required"` // 锁名称
		Desc    string `form:"desc" binding:"required"` // 锁的描述信息
		Mac     string `form:"mac" binding:"required"`
		Model   string `form:"model"`                  // 硬件型号
		Version string `form:"version"`                // 软件版本
		Key     string `form:"key" binding:"required"` // 开锁AES密钥

	}{}
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)
	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)

	newLock := model.Lock{
		Name:       params.Name,
		Desc:       params.Desc,
		Mac:        params.Mac,
		Model:      params.Model,
		Version:    params.Version,
		Key:        params.Key,
		Valid:      true,
		Own:        bson.ObjectIdHex(userId),
		UpdateTime: time.Now().Local(),
		CreateTime: time.Now().Local(),
	}
	err := lockColl.Insert(&newLock)
	if err != nil {
		utils.ResponseError(utils.PARAM_ERR, err.Error(), c)
		return
	}
	utils.ResponseOk("ok", c)
}

func UpdateLock(c *gin.Context) {
	userId := c.GetString("id")
	// 请求参数列表
	params := &struct {
		Name string `form:"name"`
		Mac  string `binding:"required"`
		Desc string `form:"desc"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}

	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)

	updateVal := &struct {
		Name       string    `bson:"name,omitempty"`
		Desc       string    `bson:"desc,omitempty"`
		UpdateTime time.Time `bson:"updateTime"` // 更新时间
	}{
		UpdateTime: time.Now().Local(),
	}
	// 只可以修改属于自己的并且没有被删除的锁
	if err := lockColl.Update(bson.M{
		"mac":   params.Mac,
		"own":   bson.ObjectIdHex(userId),
		"valid": true,
	}, bson.M{
		"$set": updateVal,
	}); err != nil {
		if err != mgo.ErrNotFound {
			utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
			return
		}
		utils.ResponseError(utils.NOT_EXISTS, "此锁不属于您或者已经被删除", c)
		return
	}

	utils.ResponseOk(fmt.Sprintf("lock[%s] update success", params.Mac), c)
}

func DeleteLock(c *gin.Context) {
	userId := c.GetString("id")
	// 请求参数列表
	params := &struct {
		Mac string `binding:"required"`
	}{}

	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)

	updateVal := &struct {
		Valid      bool      `bson:"valid"`
		UpdateTime time.Time `bson:"updateTime"` // 更新时间
	}{
		Valid:      false,
		UpdateTime: time.Now().Local(),
	}

	if err := lockColl.Update(bson.M{
		"mac": params.Mac,
		"own": bson.ObjectIdHex(userId),
	}, bson.M{
		"$set": updateVal,
	}); err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	utils.ResponseOk(fmt.Sprintf("lock[%s] delete success", params.Mac), c)
}
