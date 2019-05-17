package controller

import (
	"ezlock/common/mongo"
	"ezlock/config"
	"ezlock/middleware"
	"ezlock/model"
	"ezlock/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/medivhzhan/weapp"
	"time"
)

// 登录逻辑
func Login(c *gin.Context) {
	// code 用户登录凭证,必传
	params := &struct {
		Code          string `form:"code" binding:"required"`
		Iv            string `form:"iv" json:"iv" binding:"required"`
		EncryptedData string `form:"encryptedData" json:"encryptedData" binding:"required"`
		RawData       string `form:"rawData" json:"rawData" binding:"required"`
		Signature     string `form:"signature" json:"signature" binding:"required"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}

	res, err := weapp.Login(config.AppID, config.Secret, params.Code)
	if err != nil {
		utils.ResponseError(utils.WEAPP_ERR, err.Error(), c)
		return
	}
	//
	// 获取到用户到openid
	openId := res.OpenID
	sessionKey := res.SessionKey

	// 获取用户信息
	userInfo, err := weapp.DecryptUserInfo(params.RawData, params.EncryptedData, params.Signature, params.Iv, sessionKey)
	if err != nil {
		utils.ResponseError(utils.WEAPP_ERR, err.Error(), c)
		return
	}
	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	userColl := mgoSession.DB(config.DataBaseName).C(model.UserTableName)

	q := bson.M{
		"openId": openId,
	}
	user := model.User{}

	err = userColl.Find(q).One(&user)
	if err != nil {
		if err != mgo.ErrNotFound {
			utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
			return
		}
		user = model.User{
			Id:         bson.NewObjectId(),
			OpenId:     openId,
			SessionKey: sessionKey,
			NickName:   userInfo.Nickname,
			UnionId:    userInfo.UnionID,
			Gender:     userInfo.Gender,
			Province:   userInfo.Province,
			City:       userInfo.Province,
			Country:    userInfo.Country,
			AvatarUrl:  userInfo.Avatar,
			Language:   user.Language,
			UpdateTime: time.Now().Local(),
			CreateTime: time.Now().Local(),
		}
		if err := userColl.Insert(&user); err != nil {
			utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
			return
		}
	}

	// 更新sessionKey
	updateVal := bson.M{
		"sessionKey": sessionKey,
		"nickName":   userInfo.Nickname,
		"unionId":    userInfo.UnionID,
		"gender":     userInfo.Gender,
		"province":   userInfo.Province,
		"city":       userInfo.City,
		"country":    userInfo.Country,
		"avatarUrl":  userInfo.Avatar,
		"language":   userInfo.Language,
		"updateTime": time.Now().Local(),
	}

	if err := userColl.Update(q, bson.M{
		"$set": updateVal,
	}); err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	// 如果查询到了就获取用户id，放入jwt
	token, _, err := middleware.CreateToken(fmt.Sprintf("%s", user.Id.Hex()))
	if err != nil {
		utils.ResponseError(utils.UNAUTH, err.Error(), c)
		return
	}
	utils.ResponseOk(token, c)
}

// 获取手机号
func GetPhone(c *gin.Context) {
	userId := c.GetString("id")
	fmt.Println(userId)

	params := &struct {
		Iv            string `form:"iv" json:"iv" binding:"required"`
		EncryptedData string `form:"encryptedData" json:"encryptedData" binding:"required"`
	}{}

	if ok := utils.CheckParam(params, c); !ok {
		return
	}

	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	userColl := mgoSession.DB(config.DataBaseName).C(model.UserTableName)
	user := model.User{}

	err := userColl.FindId(userId).One(&user)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	phone, err := weapp.DecryptPhoneNumber(user.SessionKey, params.EncryptedData, params.Iv)
	if err != nil {
		utils.ResponseError(utils.WEAPP_ERR, err.Error(), c)
		return
	}

	q := bson.M{"_id": bson.ObjectIdHex(userId)}
	// 更新手机号字段
	if err := userColl.Update(q, bson.M{
		"$set": bson.M{"phone": phone, "updateTime": time.Now().Local()},
	}); err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	utils.ResponseOk(phone, c)
}

// 获取用户信息
func GetUserInfo(c *gin.Context) {
	userId := c.GetString("id")

	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	userColl := mgoSession.DB(config.DataBaseName).C(model.UserTableName)

	resp := &struct {
		NickName    string `json:"nickName" bson:"nickName"`
		PhoneNumber string `json:"phoneNumber" bson:"phoneNumber"`
		Gender      int    `json:"gender" bson:"gender"`
		Province    string `json:"province" bson:"province"`
		City        string `json:"city" bson:"city"`
		Country     string `json:"country" bson:"country"`
		AvatarUrl   string `json:"avatarUrl" bson:"avatarUrl"`
		Language    string `json:"language" bson:"language"`
	}{}

	err := userColl.FindId(userId).One(resp)
	if err != nil {
		utils.ResponseError(utils.MONGO_ERR, err.Error(), c)
		return
	}

	// 响应给前端
	utils.ResponseOk(resp, c)
}
