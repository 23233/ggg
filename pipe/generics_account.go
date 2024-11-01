package pipe

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/23233/ggg/ut"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

type AccountPlatformPublic struct {
	Favorite    int    `json:"favorite,omitempty" bson:"favorite,omitempty" comment:"我的收藏数量"` // 用于我的收藏数量
	Digg        int    `json:"digg,omitempty" bson:"digg,omitempty" comment:"被点赞数量"`          // 用于我的作品的被点赞数量
	WorkCount   int    `json:"work_count,omitempty" bson:"work_count,omitempty" comment:"作品数量"`
	Fans        int    `json:"fans,omitempty" bson:"fans,omitempty" comment:"粉丝数量"`
	Follow      int    `json:"follow,omitempty" bson:"follow,omitempty" comment:"关注数量"`
	Share       int    `json:"share,omitempty" bson:"share,omitempty" comment:"分享数量"`
	PlayView    int    `json:"play_view,omitempty" bson:"play_view,omitempty" comment:"播放数量"` // 用于被多少人点开看了
	Recommend   int    `json:"recommend,omitempty" bson:"recommend,omitempty" comment:"推荐数量"` // 用户被推荐给多少人看了
	Description string `json:"description,omitempty" bson:"description,omitempty" comment:"描述简介"`
}

type AccountPlatform struct {
	Appid      string                 `json:"appid,omitempty" bson:"appid,omitempty" comment:"平台appid"`    // 同样是微信小程序 可能有多个应用 这就是应用id
	Name       string                 `json:"name,omitempty" bson:"name,omitempty" comment:"平台名称"`         // 微信小程序 抖音
	Pid        string                 `json:"pid,omitempty" bson:"pid,omitempty" comment:"平台ID"`           // 常见于微信的openid
	UnionId    string                 `json:"union_id,omitempty" bson:"union_id,omitempty" comment:"通用ID"` // 常见于微信的unionid
	NickName   string                 `json:"nick_name,omitempty" bson:"nick_name,omitempty" comment:"平台昵称"`
	AvatarUrl  string                 `json:"avatar_url,omitempty" bson:"avatar_url,omitempty" comment:"平台头像"`
	Password   string                 `json:"password,omitempty" bson:"password,omitempty" comment:"密码"`
	Phone      string                 `json:"phone,omitempty" bson:"phone,omitempty" comment:"手机号"`                // 平台绑定的手机号码 与用户手机号码可能不同
	Scopes     []string               `json:"scopes,omitempty" bson:"scopes,omitempty" comment:"授权范围"`             // 平台的授权范围
	AuthorTime time.Time              `json:"author_time,omitempty" bson:"author_time,omitempty" comment:"平台授权时间"` // 如果是第三方授权 则有一个授权时间
	PublicData *AccountPlatformPublic `json:"public_data,omitempty" bson:"public_data,omitempty" comment:"平台公共数据"`
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
	return MongoIterateByBatch[*GenericsAccount](ctx, db, bson.M{}, nil, batchSize, processFunc)
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
