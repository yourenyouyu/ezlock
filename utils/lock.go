package utils

import (
	"ezlock/common/mongo"
	"ezlock/config"
	"ezlock/model"
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"time"
)

// 获取给定用户被授权的锁 valid 为true 在有效期限内，false就是所有
func GetAuthLocks(userId string, valid bool, perms model.Perms) ([]bson.ObjectId, error) {
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)
	authColl := mgoSession.DB(config.DataBaseName).C(model.AuthTableName)

	q := bson.M{
		"receiverId": userId,
	}
	if valid {
		q["valid"] = true
	}
	if perms.AddCard {
		q["addCard"] = true
	}
	if perms.ShareAuth {
		q["shareAuth"] = true
	}
	if perms.ViewLog {
		q["viewLog"] = true
	}
	auths := []model.Auth{}
	lockIds := []bson.ObjectId{}
	// 找到用户被授权的记录
	err := authColl.Find(q).All(&auths)
	if err != nil {
		if err != mgo.ErrNotFound {
			return nil, err
		}
		return lockIds, nil
	}

	for _, auth := range auths {
		// 授权无效
		if !CheckAuthValid(auth) {
			// 发现授权已经不在有效期内 则更新一下数据库 设置授权无效
			err := authColl.UpdateId(auth.Id, bson.M{
				"$set": bson.M{"valid": false},
			})
			if err != nil {
				return nil, err
			}
			if valid {
				continue
			}
		}
		lockIds = append(lockIds, auth.LockId)
	}
	return lockIds, nil
}

// 获取给定用户自己拥有的锁
func GetOwnLocks(userId string, valid bool) ([]bson.ObjectId, error) {
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)
	q := bson.M{
		"own": bson.ObjectIdHex(userId),
	}
	if valid {
		q["valid"] = true
	}
	locks := []model.Lock{}
	lockIds := []bson.ObjectId{}
	// 找到用户被授权的记录
	err := lockColl.Find(q).Select(bson.M{"_id": 1}).All(&locks)
	if err != nil {
		if err != mgo.ErrNotFound {
			return nil, err
		}
		return lockIds, nil
	}
	for _, lock := range locks {
		lockIds = append(lockIds, lock.Id)
	}
	return lockIds, nil
}

// 获取用户目前可用的锁
func GetAllLocks(userId string, valid bool, perms model.Perms) (map[bson.ObjectId]bool, error) {
	allLocks := map[bson.ObjectId]bool{}
	ownLocks, err := GetOwnLocks(userId, valid)
	if err != nil {
		return nil, err
	}

	for _, lock := range ownLocks {
		allLocks[lock] = true
	}

	authLocks, err := GetAuthLocks(userId, valid, perms)
	if err != nil {
		return nil, err
	}

	for _, lock := range authLocks {
		allLocks[lock] = false
	}

	return allLocks, nil
}

func GenerateKey(userId, mac, operate, code string) (key string, err error) {
	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)

	// 查看用户被授权的锁
	perms := model.Perms{}
	if operate == config.AddCard {
		perms.AddCard = true
	}

	if operate == config.AddCard {
		perms.ViewLog = true
	}

	locks, err := GetAllLocks(userId, true, perms)
	if err != nil {
		return "", err
	}

	lockIds := make([]bson.ObjectId, len(locks))
	for key := range locks {
		lockIds = append(lockIds, key)
	}
	q := bson.M{
		"_id": bson.M{"$in": lockIds},
		"mac": mac,
	}
	lock := model.Lock{}
	// 响应给用户的结构
	err = lockColl.Find(q).Select(bson.M{"_id": 0, "key": 1}).One(&lock)
	if err != nil {
		return "", err
	}
	// 如果是开门/添加门卡操作 硬件需要将如下指令格式写入日志，其他操作不写日志
	// 格式 操作指令_锁的mac地址_方式_卡号/操作用户：结果
	// 指令 %s_%s_%s 操作指令_锁的mac地址_操作用户，硬件需要写入到日志
	// 如果用户刷卡 硬件写入 操作指令_锁的mac地址_卡号 日志
	rawData := []byte(fmt.Sprintf("%s_%s_%s", operate, time.Now().Local().Format("2006-01-02 15:04"), userId))
	key, err = Encrypt([]byte(code), rawData, []byte(lock.Key))
	if err != nil {
		return "", err
	}
	return key, nil
}

func DncryptData(userId, mac, rawData string) (content string, err error) {
	// 获取mongo的操作session
	mgoSession := mongo.GetMgoSession()
	defer mongo.PutMgoSession(mgoSession)

	lockColl := mgoSession.DB(config.DataBaseName).C(model.LockTableName)

	// 查看用户被授权的锁
	locks, err := GetAllLocks(userId, true, model.Perms{})
	if err != nil {
		return "", err
	}

	lockIds := make([]bson.ObjectId, len(locks))
	for key := range locks {
		lockIds = append(lockIds, key)
	}
	q := bson.M{
		"_id": bson.M{"$in": lockIds},
		"mac": mac,
	}
	lock := model.Lock{}
	// 响应给用户的结构
	err = lockColl.Find(q).Select(bson.M{"_id": 0, "key": 1}).One(&lock)
	if err != nil {
		return "", err
	}
	content, err = Dncrypt(rawData, []byte(lock.Key))
	if err != nil {
		return "", err
	}
	return content, nil
}
