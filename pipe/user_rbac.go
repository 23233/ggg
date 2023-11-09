package pipe

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/util"
	redisadapter "github.com/casbin/redis-adapter/v2"
)

//p, admin, domain1, data1, read
//p, admin, domain1, data1, write
//p, admin, domain2, data2, read
//p, admin, domain2, data2, write
//
//g, alice, admin, *
//g, bob, admin, domain2

// 支持通配符
//p, alice, book_group, read
//g, /book/:board, book_group
//g, resource 或者 userId 均支持通配符, domain也支持通配符

// 规则与规则之间 需要有一个关系

const (
	// 默认代表所有域名的通配符
	RbacAllDomainDefault = "*"
	// 默认自身站点代号 可用于站点登录限制等
	RbacSelfDomainName = "self"
	// 登录
	RbacNotAllowLoginObj = "login"
	// 默认操作act
	RbacNormalAct = "POST"
)

type RbacDomain struct {
	E             *casbin.Enforcer
	defaultDomain string
}

func NewRbacDomain(redisAddress, password string) (*RbacDomain, error) {
	var md = new(RbacDomain)
	md.defaultDomain = RbacSelfDomainName
	err := md.initConnect(redisAddress, password)
	return md, err
}

func (c *RbacDomain) initConnect(address, password string) error {

	// 参考
	// https://github.com/casbin/casbin/blob/master/examples/keymatch_policy.csv
	// https://casbin.org/zh/docs/rbac
	// https://casbin.org/zh/docs/function

	a := redisadapter.NewAdpaterWithOption(
		redisadapter.WithNetwork("tcp"),
		redisadapter.WithAddress(address),
		redisadapter.WithPassword(password),
		redisadapter.WithKey("cas_rbac"),
	)

	m := model.NewModel()
	// 在新版本中 `dom` 可以放置在任意位置
	// request_definition
	m.AddDef("r", "r", "sub, dom, obj, act")
	// policy_definition
	// p, sub, domain1, data1/:test, read,allow | deny
	// p, book_group, domain1, data1/*, read ,allow | deny
	// p, book_group, domain1, data1/*, (GET)|(POST) ,allow | deny
	// p, book_group, domain1, data1/*, *,allow | deny
	m.AddDef("p", "p", "sub, dom, obj, act, eft")
	// role_definition
	// nowUser,group,domain 而policy中的 sub 必须为 group 指定的值即可
	// 继承角色 前项继承后项
	// 下方启用了 KeyMatch2 的正则
	m.AddDef("g", "g", "_, _, _")
	// policy_effect
	// 拒绝优先
	m.AddDef("e", "e", "some(where (p.eft == allow)) && !some(where (p.eft == deny))")
	// matchers
	// police 的obj 和 act 均支持 KeyMatch2 的正则
	// 但是 dom 尽量指定
	m.AddDef("m", "m", `g(r.sub, p.sub, r.dom) && keyMatch2(r.dom, p.dom) && keyMatch2(r.obj, p.obj) && keyMatch2(r.act, p.act)`)
	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		return err
	}
	// 看清楚这里 KeyMatch2 所以对应的关系是 * 号 和 :号
	// https://casbin.org/zh/docs/function
	// g, /book/:board, book_group, *
	// 启用了 root 超级用户
	e.AddNamedMatchingFunc("g", "KeyMatch2", util.KeyMatch2)
	e.AddNamedDomainMatchingFunc("g", "KeyMatch2", util.KeyMatch2)
	c.E = e

	// 进行预设
	err = c.presets()
	if err != nil {
		return err
	}

	return nil
}

func (c *RbacDomain) presets() error {
	// sub, dom, obj, act, eft
	_, err := c.E.AddPolicy(c.allow("root", RbacAllDomainDefault, "*", "*"))
	if err != nil {
		return err
	}
	// 职工只是没有用户管理权
	_, err = c.E.AddPolicy(c.allow("staff", RbacAllDomainDefault, "*", "*"))
	if err != nil {
		return err
	}
	// 观摩团 仅有阅读权限
	_, err = c.E.AddPolicy(c.allow("read", RbacAllDomainDefault, "*", "GET"))
	if err != nil {
		return err
	}
	// 职工不允许有 用户模型 的操作
	_, err = c.E.AddPolicy(c.localDenySelf("staff", "*", "*"))
	if err != nil {
		return err
	}
	// 观摩团也不允许有用户模型
	_, err = c.E.AddPolicy(c.localDenySelf("read", "*", "*"))
	if err != nil {
		return err
	}

	// 不允许登录 则把用户uid加入这个组中 not_login
	_, err = c.E.AddPolicy(c.localDenySelf("not_login", RbacNotAllowLoginObj, RbacNormalAct))
	if err != nil {
		return err
	}

	return nil
}

func (c *RbacDomain) localDenySelf(sub, obj, act string) []string {
	return c.deny(sub, c.defaultDomain, obj, act)
}
func (c *RbacDomain) localDeny(sub, dom, obj, act string) []string {
	return c.deny(sub, dom, obj, act)
}

func (c *RbacDomain) localAllow(sub, obj, act string) []string {
	return c.allow(sub, c.defaultDomain, obj, act)
}
func (c *RbacDomain) deny(sub, dom, obj, act string) []string {
	return []string{sub, dom, obj, act, "deny"}
}
func (c *RbacDomain) allow(sub, dom, obj, act string) []string {
	return []string{sub, dom, obj, act, "allow"}
}

// 对于身份的设置

func (c *RbacDomain) SetRoot(uid string) (bool, error) {
	return c.SetRoleDefaultDomain(uid, "root")
}
func (c *RbacDomain) DelRoot(uid string) error {

	_, err := c.DelRoleDefaultDomain(uid, "root")
	return err
}

func (c *RbacDomain) SetRole(uid string, role string, domain string) (bool, error) {
	return c.E.AddRoleForUser(uid, role, domain)
}
func (c *RbacDomain) SetRoleDefaultDomain(uid string, role string) (bool, error) {
	return c.E.AddRoleForUser(uid, role, RbacAllDomainDefault)
}

func (c *RbacDomain) DelRole(uid string, role string, domain string) (bool, error) {
	return c.E.DeleteRoleForUserInDomain(uid, role, domain)
}
func (c *RbacDomain) DelRoleDefaultDomain(uid string, role string) (bool, error) {
	return c.E.DeleteRoleForUserInDomain(uid, role, RbacAllDomainDefault)
}

func (c *RbacDomain) SetStaff(uid string) (bool, error) {
	return c.SetRoleDefaultDomain(uid, "staff")
}
func (c *RbacDomain) DelStaff(uid string) error {
	_, err := c.DelRoleDefaultDomain(uid, "staff")
	return err
}

func (c *RbacDomain) SetRead(uid string) (bool, error) {
	// 当设置仅只读时 其他身份最好去掉
	return c.E.AddRoleForUser(uid, "read", RbacAllDomainDefault)
}
func (c *RbacDomain) DelRead(uid string) error {
	_, err := c.E.DeleteRoleForUserInDomain(uid, "read", RbacAllDomainDefault)
	return err
}

func (c *RbacDomain) IsStaffOrRoot(uid string) bool {
	return c.HasRoles(uid, []string{"root", "staff"})
}

func (c *RbacDomain) HasRoles(uid string, roles []string) bool {
	domainRoles := c.E.GetRolesForUserInDomain(uid, RbacAllDomainDefault)

	for _, role := range domainRoles {
		for _, r := range roles {
			if r == role {
				return true
			}
		}
	}

	return false
}
