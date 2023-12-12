package scene

import (
	"github.com/23233/ggg/ut"
	"github.com/qiniu/qmgo"
)

func NewItem(db *qmgo.Database, tableName string, inputStruct any, f ItemHandler) (ISceneModelItem, error) {
	if len(tableName) < 1 {
		return nil, ErrTableNameEmpty
	}
	var item = new(Item)
	item.SetTableName(tableName)
	item.SetSchema(inputStruct)
	item.SetHandler(f)
	item.SetDb(db)
	item.SetRaw(inputStruct)
	item.extra = new(ItemExtra)
	return item, nil
}
func NewGetAllItem(db *qmgo.Database, tableName string, inputStruct any, contextInjectToFilter []ContextValueInject, injectsToFilter []*ut.Kov, getCount bool, mustLastId bool) (ISceneModelItem, error) {
	item, err := NewItem(db, tableName, inputStruct, FuncGetAll)
	item.SetContextInjectFilter(contextInjectToFilter)
	item.SetFilterInjects(injectsToFilter)
	item.GetExtra().GetCount = getCount
	item.GetExtra().MustLastId = mustLastId
	return item, err
}
func NewGetSingleItem(db *qmgo.Database, tableName string, inputStruct any, contextInjectToFilter []ContextValueInject, injectsToFilter []*ut.Kov) (ISceneModelItem, error) {
	item, err := NewItem(db, tableName, inputStruct, FuncGetSingle)
	item.SetContextInjectFilter(contextInjectToFilter)
	item.SetFilterInjects(injectsToFilter)
	return item, err
}
func NewAddItem(db *qmgo.Database, tableName string, inputStruct any, contextInjectToBody []ContextValueInject, injectsToBody []*ut.Kov) (ISceneModelItem, error) {
	item, err := NewItem(db, tableName, inputStruct, FuncPostAdd)
	item.SetContextInjectFilter(contextInjectToBody)
	item.SetFilterInjects(injectsToBody)
	return item, err
}
func NewEditItem(db *qmgo.Database, tableName string, inputStruct any, contextInjectToFilter []ContextValueInject, injectsToFilter []*ut.Kov) (ISceneModelItem, error) {
	item, err := NewItem(db, tableName, inputStruct, FuncEdit)
	item.SetContextInjectFilter(contextInjectToFilter)
	item.SetFilterInjects(injectsToFilter)
	return item, err
}
func NewDeleteItem(db *qmgo.Database, tableName string, inputStruct any, contextInjectToFilter []ContextValueInject, injectsToFilter []*ut.Kov) (ISceneModelItem, error) {
	item, err := NewItem(db, tableName, inputStruct, FuncDelete)
	item.SetContextInjectFilter(contextInjectToFilter)
	item.SetFilterInjects(injectsToFilter)
	return item, err
}
