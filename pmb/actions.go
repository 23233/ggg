package pmb

import (
	"github.com/23233/ggg/ut"
	"github.com/23233/jsonschema"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
)

type ActionPostPart[F any] struct {
	Name     string   `json:"name"`                // action名称
	Rows     []string `json:"rows,omitempty"`      // 选中的行id
	FormData F        `json:"form_data,omitempty"` // 表单填写的值
}

// ActionPostArgs T是行数据 F是表单数据
type ActionPostArgs[T, F any] struct {
	Rows     []T              `json:"rows"`
	FormData F                `json:"form_data"`
	User     *SimpleUserModel `json:"user"`
	Model    IModelItem       `json:"model"`
}

type SchemaActionBase struct {
	Name                        string             `json:"name,omitempty"`                   // 动作名称 需要唯一
	Prefix                      string             `json:"prefix,omitempty"`                 // 前缀标识 仅展示用
	Types                       []uint             `json:"types,omitempty"`                  // 0 表可用 1 行可用
	Form                        *jsonschema.Schema `json:"form,omitempty"`                   // 若form为nil 则不会弹出表单填写
	MustSelect                  bool               `json:"must_select,omitempty"`            // 必须有所选择表选择适用 行是必须选一行
	TableEmptySelectUseAllSheet bool               `json:"table_empty_select_use_all_sheet"` // 表模式未选中行则默认是整表
	Conditions                  []ut.Kov           `json:"conditions,omitempty"`             // 选中/执行的前置条件 判断数据为选中的每一行数据 常用场景为 限定只有字段a=b时才可用或a!=b时 挨个执行 任意一个不成功都返回
	FaWarning                   bool               `json:"fa_warning,omitempty"`             // 是否弹出二次确认操作
}

func (s *SchemaActionBase) SetType(tp []uint) {
	s.Types = tp
}
func (s *SchemaActionBase) SetForm(raw any) {
	s.Form = ToJsonSchema(raw)
}
func (s *SchemaActionBase) AddCondition(cond ut.Kov) {
	s.Conditions = append(s.Conditions, cond)
}

type ISchemaAction interface {
	Execute(ctx iris.Context, args any) (responseInfo any, err error)
	GetBase() *SchemaActionBase
	SetCall(func(ctx iris.Context, args any) (responseInfo any, err error))
}

func ActionParseArgs[T any, F any](ctx iris.Context, model IModelItem) (*ActionPostPart[F], []T, ISchemaAction, error) {
	// 必须为post
	part := new(ActionPostPart[F])
	err := ctx.ReadBody(&part)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "解构action参数包失败")
	}

	action, has := model.GetAction(part.Name)
	if !has {
		return nil, nil, nil, errors.Wrap(err, "未找到对应action")

	}

	rows := make([]T, 0, len(part.Rows))
	if len(part.Rows) >= 1 {
		// 去获取出最新的这一批数据
		err = model.GetCollection().Find(ctx, bson.M{ut.DefaultUidTag: bson.M{"$in": part.Rows}}).All(&rows)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "获取对应行列表失败")
		}
	}
	return part, rows, action, nil
}

func ActionRun[T any, F any](ctx iris.Context, model IModelItem, user *SimpleUserModel) {
	part, rows, action, err := ActionParseArgs[T, F](ctx, model)
	args := new(ActionPostArgs[T, F])
	args.Rows = rows
	args.FormData = part.FormData
	args.User = user
	args.Model = model

	result, err := action.Execute(ctx, args)
	if err != nil {
		IrisRespErr("", err, ctx)
		return
	}

	if result != nil {
		_ = ctx.JSON(result)
		return
	}
	_, _ = ctx.WriteString("ok")

}

// SchemaModelAction 模型action T是模型数据 F是表单数据
type SchemaModelAction[T, F any] struct {
	SchemaActionBase
	call func(ctx iris.Context, args any) (responseInfo any, err error)
}

func (s *SchemaModelAction[T, F]) GetBase() *SchemaActionBase {
	return &s.SchemaActionBase
}
func (s *SchemaModelAction[T, F]) Execute(ctx iris.Context, args any) (responseInfo any, err error) {
	if s.call == nil {
		return nil, errors.New("action未定义默认执行函数")
	}
	return s.call(ctx, args)
}
func (s *SchemaModelAction[T, F]) SetCall(call func(ctx iris.Context, args any) (responseInfo any, err error)) {
	s.call = call
}

func NewSchemaModelAction[T, F any]() *SchemaModelAction[T, F] {
	inst := new(SchemaModelAction[T, F])
	return inst
}

func NewRowAction[T any, F any](name string, form F) ISchemaAction {
	inst := NewSchemaModelAction[T, F]()
	inst.Types = []uint{1}
	inst.Name = name

	val := reflect.ValueOf(form)
	if val.Kind() == reflect.Ptr {
		if !val.IsNil() {
			inst.SetForm(form)
		}
	} else if val.IsValid() && !val.IsZero() {
		inst.SetForm(form)
	}
	inst.Conditions = make([]ut.Kov, 0)
	return inst
}
func NewTableAction[T any, F any](name string, form F) ISchemaAction {
	inst := NewRowAction[T, F](name, form)
	inst.GetBase().SetType([]uint{0})
	return inst
}

// NewAction action的名称必填 form没有可传nil
func NewAction[T any, F any](name string, form F) ISchemaAction {
	inst := NewRowAction[T, F](name, form)
	inst.GetBase().SetType([]uint{0, 1})
	return inst
}
