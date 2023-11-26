package pipe

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/23233/ggg/logger"
	"github.com/23233/ggg/ut"
	"github.com/qiniu/qmgo"
	opta "github.com/qiniu/qmgo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AccountPlatform struct {
	Name      string `json:"name,omitempty" bson:"name,omitempty" comment:"平台名称"`         // 微信小程序
	Pid       string `json:"pid,omitempty" bson:"pid,omitempty" comment:"平台ID"`           // 常见于微信的openid
	UnionId   string `json:"union_id,omitempty" bson:"union_id,omitempty" comment:"通用ID"` // 常见于微信的unionid
	NickName  string `json:"nick_name,omitempty" bson:"nick_name,omitempty" comment:"平台昵称"`
	AvatarUrl string `json:"avatar_url,omitempty" bson:"avatar_url,omitempty" comment:"平台头像"`
	Password  string `json:"password,omitempty" bson:"password,omitempty" comment:"密码"`
	Data      string `json:"data,omitempty" bson:"data,omitempty" comment:"平台额外数据"`
}

type AccountPass struct {
	UserName string `json:"user_name,omitempty" bson:"user_name,omitempty" comment:"用户名"`
	Password string `json:"password,omitempty" bson:"password,omitempty" comment:"加密后登录密码"`
	Salt     string `json:"salt,omitempty" bson:"salt,omitempty" comment:"密码加密salt"`
	Email    string `json:"email,omitempty" bson:"email,omitempty" comment:"邮箱地址"`
	TelPhone string `json:"tel_phone,omitempty" bson:"tel_phone,omitempty" comment:"电话号码"` // 电话号码
}

func (a *AccountPass) SetUserName(newUserName string) {
	a.UserName = newUserName
}
func (a *AccountPass) GetUserName() string {
	return a.UserName
}

// SetPassword 设置密码 明文 会自动加密
func (a *AccountPass) SetPassword(newPassword string) {
	m5ps, salt := a.PasswordMd5(newPassword)
	a.Password = m5ps
	a.Salt = salt
}
func (a *AccountPass) GetPassword() string {
	return a.Password
}

func (a *AccountPass) SetEmail(newEmail string) {
	a.Email = newEmail
}
func (a *AccountPass) GetEmail() string {
	return a.Email
}
func (a *AccountPass) SetTelPhone(newTelPhone string) {
	a.TelPhone = newTelPhone
}
func (a *AccountPass) GetTelPhone() string {
	return a.TelPhone
}

// PasswordMd5 通过原始密码生成加密的m5为password
func (a *AccountPass) PasswordMd5(rawPassword string) (m5ps string, salt string) {
	if len(a.Salt) < 1 {
		a.Salt = ut.RandomStr(4)
	}
	m5 := md5.New()
	m5.Write([]byte(rawPassword))
	m5.Write([]byte(a.Salt))
	st := m5.Sum(nil)
	m5ps = hex.EncodeToString(st)
	return m5ps, a.Salt
}

// ValidPassword 验证密码输出是否正确 password 为输入密码
func (a *AccountPass) ValidPassword(rawPassword string) bool {
	r := md5.New()
	r.Write([]byte(rawPassword))
	r.Write([]byte(a.Salt))
	st := r.Sum(nil)
	ps := hex.EncodeToString(st)
	return ps == a.Password
}

type AccountCoin struct {
	Balance     uint64 `json:"balance,omitempty" bson:"balance,omitempty" comment:"余额(分)" `           // 余额 单位是分
	ReferrerUid string `json:"referrer_uid,omitempty" bson:"referrer_uid,omitempty" comment:"介绍人uid"` // 介绍人
}

func (a *AccountCoin) GetBalance() uint64 {
	return a.Balance
}

func (a *AccountCoin) SetBalance(newBalance uint64) {
	a.Balance = newBalance
}

func (a *AccountCoin) GetReferrerUid() string {
	return a.ReferrerUid
}

func (a *AccountCoin) SetReferrerUid(referrer string) {
	a.ReferrerUid = referrer
}

type AccountComm struct {
	AvatarUrl string `json:"avatar_url,omitempty" bson:"avatar_url,omitempty" comment:"头像地址"` // 头像地址
	NickName  string `json:"nick_name,omitempty" bson:"nick_name,omitempty" comment:"昵称"`     // 昵称
	Disable   bool   `json:"disable,omitempty" bson:"disable" comment:"是否禁用"`
	Msg       string `json:"msg,omitempty" bson:"msg,omitempty" comment:"状态说明"`
}

func (a *AccountComm) GetAvatarUrl() string {
	return a.AvatarUrl
}

func (a *AccountComm) SetAvatarUrl(uri string) {
	a.AvatarUrl = uri
}

func (a *AccountComm) GetNickName() string {
	return a.NickName
}

func (a *AccountComm) SetNickName(name string) {
	a.NickName = name
}

func (a *AccountComm) GetMsg() string {
	return a.Msg
}

func (a *AccountComm) SetMsg(newMsg string) {
	a.Msg = newMsg
}

func (a *AccountComm) GetDisable() bool {
	return a.Disable
}

func (a *AccountComm) SetDisable(newDisable bool) {
	a.Disable = newDisable
}

type GenericsAccount struct {
	ModelBase   `bson:",inline"`
	AccountPass `bson:",inline"`
	AccountCoin `bson:",inline"`
	AccountComm `bson:",inline"`
	// 在mongo中 可以直接使用.法 也就是 platforms.name 就可以传所有数组对象的name包含了的
	Platforms []*AccountPlatform `json:"platforms,omitempty" bson:"platforms,omitempty" comment:"平台信息"`
}

func (s *GenericsAccount) SetAccountPass(pass AccountPass) {
	s.AccountPass = pass
}

func (s *GenericsAccount) GetAccountPass() AccountPass {
	return s.AccountPass
}

func (s *GenericsAccount) GetCoin() AccountCoin {
	return s.AccountCoin
}

func (s *GenericsAccount) setCoin(coin AccountCoin) {
	s.AccountCoin = coin
}

func (s *GenericsAccount) GetComm() AccountComm {
	return s.AccountComm
}

func (s *GenericsAccount) SetComm(comm AccountComm) {
	s.AccountComm = comm
}

func (s *GenericsAccount) Filters(ctx context.Context, db *qmgo.Collection, filters bson.M) ([]*GenericsAccount, error) {
	return MongoFilters[*GenericsAccount](ctx, db, filters)
}
func (s *GenericsAccount) GetOne(ctx context.Context, db *qmgo.Collection, uid string) (*GenericsAccount, error) {
	return MongoGetOne[GenericsAccount](ctx, db, uid)
}
func (s *GenericsAccount) Random(ctx context.Context, db *qmgo.Collection, filters bson.D, count int) ([]*GenericsAccount, error) {
	return MongoRandom[*GenericsAccount](ctx, db, filters, count)
}
func (s *GenericsAccount) UpdateOne(ctx context.Context, db *qmgo.Collection, uid string, pack bson.M) error {
	return MongoUpdateOne(ctx, db, uid, pack)
}
func (s *GenericsAccount) iterateAccountsByBatch(ctx context.Context, db *qmgo.Collection, batchSize int64, processFunc func([]*GenericsAccount) error) error {
	return MongoIterateByBatch[*GenericsAccount](ctx, db, batchSize, processFunc)
}
func (s *GenericsAccount) BulkInsert(ctx context.Context, db *qmgo.Collection, accounts ...*GenericsAccount) error {
	return MongoBulkInsert[*GenericsAccount](ctx, db, accounts...)
}

type IAccountGenericsFull interface {
	IAccountGenerics
	IAccountShortcut
}

type IAccountGenerics interface {
	SetAccountPass(pass AccountPass)
	GetAccountPass() AccountPass
	GetCoin() AccountCoin
	setCoin(coin AccountCoin)
	GetComm() AccountComm
	SetComm(comm AccountComm)
}

type IAccountShortcut interface {
	Filters(ctx context.Context, db *qmgo.Collection, filters bson.M) ([]*GenericsAccount, error)
	GetOne(ctx context.Context, db *qmgo.Collection, uid string) (*GenericsAccount, error)
	Random(ctx context.Context, db *qmgo.Collection, filters bson.D, count int) ([]*GenericsAccount, error)
	UpdateOne(ctx context.Context, db *qmgo.Collection, uid string, pack bson.M) error
	iterateAccountsByBatch(ctx context.Context, db *qmgo.Collection, batchSize int64, processFunc func([]*GenericsAccount) error) error
	BulkInsert(ctx context.Context, db *qmgo.Collection, accounts ...*GenericsAccount) error
}

var _ IAccountGenericsFull = (*GenericsAccount)(nil)

// 封装常见操作方法

func MongoFilters[T any](ctx context.Context, db *qmgo.Collection, filters bson.M) ([]T, error) {
	var result = make([]T, 0)
	err := db.Find(ctx, filters).All(&result)
	if err != nil {
		return nil, err
	}
	return result, err
}

// MongoGetOne 传入实例 返回new之后的指针
func MongoGetOne[T any](ctx context.Context, db *qmgo.Collection, uid string) (*T, error) {
	var result = new(T)
	err := db.Find(ctx, bson.M{ut.DefaultUidTag: uid}).One(result)
	if err != nil {
		return result, err
	}
	return result, err
}
func MongoRandom[T any](ctx context.Context, db *qmgo.Collection, filters bson.D, count int) ([]T, error) {
	pipeline := mongo.Pipeline{
		{{"$match", filters}},
		{{"$sample", bson.D{{"size", count}}}},
	}
	var accounts = make([]T, 0)
	err := db.Aggregate(ctx, pipeline).All(&accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}
func MongoUpdateOne(ctx context.Context, db *qmgo.Collection, uid string, pack bson.M) error {
	return db.UpdateOne(ctx, bson.M{ut.DefaultUidTag: uid}, bson.M{"$set": pack})
}
func MongoIterateByBatch[T IMongoBase](ctx context.Context, db *qmgo.Collection, batchSize int64, processFunc func([]T) error) error {
	lastID := primitive.NilObjectID

	for {
		var accounts = make([]T, 0)

		err := db.Find(ctx, bson.M{"_id": bson.M{"$gt": lastID}}).Sort("_id").Limit(batchSize).All(&accounts)
		if err != nil {
			return fmt.Errorf("error fetching records: %v", err)
		}

		if len(accounts) == 0 {
			break
		}

		// 使用提供的处理函数处理这批数据
		err = processFunc(accounts)
		if err != nil {
			return err
		}

		// 更新 lastID 以供下一次迭代使用
		lastID = accounts[len(accounts)-1].GetBase().Id
	}

	return nil
}
func MongoBulkInsert[T any](ctx context.Context, db *qmgo.Collection, accounts ...T) error {
	// 批量新增
	opts := opta.InsertManyOptions{}
	// order默认为true 则一个报错后面都停 所以设置为false
	opts.InsertManyOptions = options.InsertMany().SetOrdered(false)

	result, err := db.InsertMany(ctx, accounts, opts)
	if err != nil {
		var bulkErr mongo.BulkWriteException
		if ok := errors.As(err, &bulkErr); !ok {
			logger.J.ErrorE(err, "批量插入发生非bulkWrite异常")
			return err
		}
		if len(result.InsertedIDs) < 1 {
			return PipeBulkEmptySuccessError
		}
	}
	logger.J.Infof("批量插入成功 %d 条", len(result.InsertedIDs))
	return nil
}
