package service

import (
	"AirGo/global"
	"AirGo/model"
	"AirGo/utils/encrypt_plugin"
	"AirGo/utils/other_plugin"
	"errors"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"time"
)

// 从v2board sspanel迁移用户数据
func Migration(mig *model.Migration) (string, error) {
	start := time.Now()
	//连接原来服务器
	var oldDB *gorm.DB
	mysqlConfig := mysql.Config{
		DSN: mig.DBUsername + ":" + mig.DBPassword + "@tcp(" + mig.DBAddress + ":" + fmt.Sprintf("%d", mig.DBPort) + ")/" + mig.DBName + "?charset=utf8mb4&parseTime=True&loc=Local",
	}
	oldDB, err := gorm.Open(mysql.New(mysqlConfig), &gorm.Config{
		SkipDefaultTransaction: true, //关闭事务
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		global.Logrus.Error("Migration gorm open error:", err)
		return "", err
	}
	//查询用户
	var userList []model.User
	switch mig.PanelType {
	case "v2board":
		err = oldDB.Table("v2_user").Select("email as user_name, uuid").Find(&userList).Error
		if err != nil {
			fmt.Println(err)
			return "", err
		}
	case "sspanel":
		err = oldDB.Debug().Table("user").Select("email as user_name, uuid, user_name as nick_name").Find(&userList).Error
		if err != nil {
			fmt.Println(err)
			return "", err
		}
	}
	//处理用户默认数据
	newUserList := UserDefaultValues(&userList)
	length := len(*newUserList)
	fmt.Println("length:", length)
	if length == 0 {
		return "", errors.New("Data is empty")
	}
	if length > 1000 {
		u := other_plugin.SplitArray[model.User](userList, int64(length/1000)+1) //防止数据过大迁移失败，分段插入
		for _, v := range u {
			err = global.DB.Create(&v).Error
			if err != nil {
				fmt.Println("Create error:", err)
				return "", err
			}
		}

	} else {
		err = global.DB.Create(&userList).Error
		if err != nil {
			fmt.Println("Create error:", err)
			return "", err
		}
	}

	end := time.Now()
	msg := fmt.Sprintf("迁移用户数据：%d 条，耗时：%s", length, end.Sub(start).String())
	return msg, nil
}

func UserDefaultValues(userList *[]model.User) *[]model.User {
	password := encrypt_plugin.BcryptEncode("123456") //迁移后默认密码
	for k, _ := range *userList {
		(*userList)[k].Password = password
		(*userList)[k].Password = password
		(*userList)[k].InvitationCode = encrypt_plugin.RandomString(8)             //邀请码
		(*userList)[k].Remain = 0                                                  //余额
		(*userList)[k].SubscribeInfo.SubscribeUrl = encrypt_plugin.RandomString(8) //随机字符串订阅url
		(*userList)[k].RoleGroup = []model.Role{{ID: 2}}                           //默认角色：普通用户角色
	}
	return userList
}
