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
	"strings"
	"time"
)

func GetAddCardKey(c *gin.Context) {
	userId := c.GetString("id")
	// 请求参数列表
	params := &struct {
		Code string `form:"code" binding:"len=16,required"`
		Mac  string `form:"mac" binding:"required"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}

	key, err := utils.GenerateKey(userId, params.Mac, config.AddCard, params.Code)
	if err != nil {
		utils.ResponseError(utils.ENCRYPT_ERR, err.Error(), c)
		return
	}

	utils.ResponseOk(key, c)
}

func GetDelCardKey(c *gin.Context) {
	userId := c.GetString("id")
	// 请求参数列表
	params := &struct {
		Code   string `form:"code" binding:"len=16,required"`
		CardId string `form:"cardId" binding:"required"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}
	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	cardColl := mgoSession.DB(config.DataBaseName).C(model.CardTableName)
	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)

	// 获取卡片所有者
	card := model.Card{}
	err := cardColl.Find(bson.M{
		"_id":   params.CardId,
		"valid": true,
	}).Select(bson.M{"userId": 1}).One(&card)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	// 获取门锁所有者
	lock := model.Lock{}
	err = lockColl.FindId(card.Lock).Select(bson.M{"own": 1}).One(&lock)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	userIds := map[bson.ObjectId]bool{
		card.UserId: true,
		lock.Own:    true,
	}
	if _, ok := userIds[bson.ObjectIdHex(userId)]; !ok {
		utils.ResponseError(utils.NOT_EXISTS, "此卡片不属于您", c)
		return
	}
	key, err := utils.GenerateKey(userId, lock.Mac, fmt.Sprintf(config.DelCard, card.Number), params.Code)
	if err != nil {
		utils.ResponseError(utils.ENCRYPT_ERR, err.Error(), c)
		return
	}

	utils.ResponseOk(key, c)
}

func GetLockCardList(c *gin.Context) {
	userId := c.GetString("id")
	params := &struct {
		Mac string `form:"mac" binding:"required"`
		//ShowValid bool   `form:"showValid"` // false 就是获取所有门卡 包括被删除的
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}
	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)
	cardColl := mgoSession.DB(config.DataBaseName).C(model.CardTableName)

	lock := model.Lock{}
	err := lockColl.Find(bson.M{
		"mac": params.Mac,
	}).Select(bson.M{"_id": 1}).One(&lock)
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

	if _, ok := locks[lock.Id]; !ok {
		utils.ResponseError(utils.UNAUTH, "您无权查看此锁的门卡信息", c)
		return
	}
	cards := []model.Card{}
	q := bson.M{"lock": lock.Id, "valid": true}
	//if params.ShowInValid {
	//	q["valid"] = true
	//}
	currentUserId := bson.ObjectIdHex(userId)
	if lock.Own != currentUserId {
		q["userId"] = currentUserId
	}
	err = cardColl.Find(q).All(&cards)
	if err != nil {
		if err != mgo.ErrNotFound {
			utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
			return
		}
	}
	utils.ResponseOk(cards, c)
}

func UpdateCard(c *gin.Context) {
	userId := c.GetString("id")
	// 请求参数列表
	params := &struct {
		CardId string `form:"cardId" binding:"required"`
		Name   string `form:"name"`
		Desc   string `form:"desc"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}

	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	cardColl := mgoSession.DB(config.DataBaseName).C(model.CardTableName)
	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)

	// 获取卡片所有者
	card := model.Card{}
	err := cardColl.Find(bson.M{"_id": params.CardId, "valid": true}).Select(bson.M{"userId": 1}).One(&card)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	// 获取门锁所有者
	lock := model.Lock{}
	err = lockColl.FindId(card.Lock).Select(bson.M{"own": 1}).One(&lock)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	updateVal := &struct {
		Name       string    `bson:"name,omitempty"`
		Desc       string    `bson:"desc,omitempty"`
		UpdateTime time.Time `bson:"updateTime"` // 更新时间
	}{
		UpdateTime: time.Now().Local(),
	}
	userIds := map[bson.ObjectId]bool{
		card.UserId: true,
		lock.Own:    true,
	}
	if _, ok := userIds[bson.ObjectIdHex(userId)]; !ok {
		utils.ResponseError(utils.NOT_EXISTS, "此卡片不属于您", c)
		return
	}
	if err := cardColl.Update(bson.M{
		"_id": bson.ObjectIdHex(params.CardId),
	}, bson.M{
		"$set": updateVal,
	}); err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return

	}

	utils.ResponseOk(fmt.Sprintf("card[%s] update success", params.Name), c)
}

// 硬件需要对锁的日志信息做个加密防止篡改，前端小程序蓝牙链接成功后 拿到这个加密信息直接发送给后端
func SetLockCard(c *gin.Context) {
	userId := c.GetString("id")
	// data 格式 cardNumber
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

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LogTableName)
	cardColl := mgoSession.DB(config.DataBaseName).C(model.CardTableName)
	userColl := mgoSession.DB(config.DataBaseName).C(model.UserTableName)

	cardNum := strings.TrimSpace(content)

	lock := model.Lock{}
	err = lockColl.Find(bson.M{"mac": params.Mac}).Select(bson.M{"_id": 1}).One(&lock)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	user := model.User{}
	err = userColl.FindId(userId).One(&user)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	// 查看这个锁有多少门禁卡 按照序号自增
	existsCards := []model.Card{}
	err = cardColl.Find(bson.M{"lock": lock.Id}).All(&existsCards)
	if err != nil {
		if err != mgo.ErrNotFound {
			utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
			return
		}
	}
	for _, card := range existsCards {
		if card.Number == cardNum {
			utils.ResponseError(utils.PARAM_ERR, "门卡已经存在", c)
			return
		}
	}
	count := len(existsCards)
	initName := fmt.Sprintf("%s的门卡 %d", user.NickName, count)
	card := model.Card{
		Name:       initName,
		Desc:       initName,
		Lock:       lock.Id,
		Number:     cardNum,
		Valid:      true,
		UserId:     bson.ObjectIdHex(userId),
		UpdateTime: time.Now().Local(),
		CreateTime: time.Now().Local(),
	}
	err = cardColl.Insert(card)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	utils.ResponseOk("ok", c)
}

// 硬件需要对锁的日志信息做个加密防止篡改，前端小程序蓝牙链接成功后 拿到这个加密信息直接发送给后端
func DelCard(c *gin.Context) {
	userId := c.GetString("id")
	// data 格式 cardNumber
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

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LogTableName)
	cardColl := mgoSession.DB(config.DataBaseName).C(model.CardTableName)

	cardNum := strings.TrimSpace(content)

	lock := model.Lock{}
	err = lockColl.Find(bson.M{"mac": params.Mac}).Select(bson.M{"_id": 1}).One(&lock)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}
	updateVal := &struct {
		Valid      bool      `bson:"valid"`
		UpdateTime time.Time `bson:"updateTime"` // 更新时间
	}{
		Valid:      false,
		UpdateTime: time.Now().Local(),
	}
	err = cardColl.Update(bson.M{"lock": lock.Id, "number": cardNum}, bson.M{
		"$set": updateVal,
	})
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	utils.ResponseOk("ok", c)
}
