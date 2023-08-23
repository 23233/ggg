package pmb

import (
	"github.com/23233/ggg/pipe"
	"github.com/qiniu/qmgo"
	"github.com/redis/rueidis"
)

// 注入的连接信息
type connectInfo struct {
	db   *qmgo.Database
	rdb  rueidis.Client
	rbac *pipe.RbacDomain
}

func (c *connectInfo) Db() *qmgo.Database {
	return c.db
}

func (c *connectInfo) AddRdb(rdb rueidis.Client) {
	c.rdb = rdb
}
func (c *connectInfo) AddDb(mdb *qmgo.Database) {
	c.db = mdb
}
func (c *connectInfo) AddRbacUseUri(redisAddress, password string) bool {
	inst, err := pipe.NewRbacDomain(redisAddress, password)
	if err != nil {
		return false
	}
	c.rbac = inst
	return false
}
func (c *connectInfo) AddRbac(rbac *pipe.RbacDomain) {
	c.rbac = rbac
}
func (c *connectInfo) SetConn(conn *connectInfo) {
	c.db = conn.db
	c.rdb = conn.rdb
	c.rbac = conn.rbac
}
func (c *connectInfo) CloneConn() *connectInfo {
	return &connectInfo{
		db:   c.db,
		rdb:  c.rdb,
		rbac: c.rbac,
	}
}
func (c *connectInfo) OpLog() *qmgo.Collection {
	return c.db.Collection("operation_log")
}
