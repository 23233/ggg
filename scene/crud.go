package scene

import (
	"github.com/23233/ggg/ut"
	"github.com/qiniu/qmgo"
)

type Crud struct {
	getAll                ISceneModelItem
	getSingle             ISceneModelItem
	add                   ISceneModelItem
	edit                  ISceneModelItem
	delete                ISceneModelItem
	contextInjectToFilter []ContextValueInject
	injectsToFilter       []*ut.Kov
	contextInjectToBody   []ContextValueInject
	injectsToBody         []*ut.Kov
	db                    *qmgo.Database
	tableName             string
	input                 any
	getCount              bool
	mustLastId            bool
}

func (c *Crud) GetCount() bool {
	return c.getCount
}

func (c *Crud) SetGetCount(getCount bool) {
	c.getCount = getCount
}

func (c *Crud) MustLastId() bool {
	return c.mustLastId
}

func (c *Crud) SetMustLastId(mustLastId bool) {
	c.mustLastId = mustLastId
}

func (c *Crud) GetAll() ISceneModelItem {
	return c.getAll
}

func (c *Crud) SetGetAll(getAll ISceneModelItem) {
	c.getAll = getAll
}

func (c *Crud) GetSingle() ISceneModelItem {
	return c.getSingle
}

func (c *Crud) SetGetSingle(getSingle ISceneModelItem) {
	c.getSingle = getSingle
}

func (c *Crud) Add() ISceneModelItem {
	return c.add
}

func (c *Crud) SetAdd(add ISceneModelItem) {
	c.add = add
}

func (c *Crud) Edit() ISceneModelItem {
	return c.edit
}

func (c *Crud) SetEdit(edit ISceneModelItem) {
	c.edit = edit
}

func (c *Crud) Delete() ISceneModelItem {
	return c.delete
}

func (c *Crud) SetDelete(delete ISceneModelItem) {
	c.delete = delete
}

func (c *Crud) ContextInjectToFilter() []ContextValueInject {
	return c.contextInjectToFilter
}

func (c *Crud) SetContextInjectToFilter(contextInjectToFilter []ContextValueInject) {
	c.contextInjectToFilter = contextInjectToFilter
}

func (c *Crud) InjectsToFilter() []*ut.Kov {
	return c.injectsToFilter
}

func (c *Crud) SetInjectsToFilter(injectsToFilter []*ut.Kov) {
	c.injectsToFilter = injectsToFilter
}

func (c *Crud) ContextInjectToBody() []ContextValueInject {
	return c.contextInjectToBody
}

func (c *Crud) SetContextInjectToBody(contextInjectToBody []ContextValueInject) {
	c.contextInjectToBody = contextInjectToBody
}

func (c *Crud) InjectsToBody() []*ut.Kov {
	return c.injectsToBody
}

func (c *Crud) SetInjectsToBody(injectsToBody []*ut.Kov) {
	c.injectsToBody = injectsToBody
}

func (c *Crud) registryToManager(m IScenes, scope string, model string) error {
	var err error
	if c.getAll == nil {
		c.getAll, err = NewGetAllItem(c.db, c.tableName, c.input, c.contextInjectToFilter, c.injectsToFilter, c.getCount, c.mustLastId)
		if err != nil {
			return err
		}
	}
	err = m.RegistryItem(scope, model, "GET_All", c.getAll)
	if err != nil {
		return err
	}

	if c.getSingle == nil {
		c.getSingle, err = NewGetSingleItem(c.db, c.tableName, c.input, c.contextInjectToFilter, c.injectsToFilter)
		if err != nil {
			return err
		}
	}
	err = m.RegistryItem(scope, model, "GET_SINGLE", c.getSingle)
	if err != nil {
		return err
	}
	if c.add == nil {
		c.add, err = NewAddItem(c.db, c.tableName, c.input, c.contextInjectToBody, c.injectsToBody)
		if err != nil {
			return err
		}
	}
	err = m.RegistryItem(scope, model, "ADD", c.add)
	if err != nil {
		return err
	}

	if c.edit == nil {
		c.edit, err = NewEditItem(c.db, c.tableName, c.input, c.contextInjectToFilter, c.injectsToFilter)
		if err != nil {
			return err
		}
	}
	err = m.RegistryItem(scope, model, "EDIT", c.edit)
	if err != nil {
		return err
	}
	if c.delete == nil {
		c.delete, err = NewDeleteItem(c.db, c.tableName, c.input, c.contextInjectToFilter, c.injectsToFilter)
		if err != nil {
			return err
		}
	}
	err = m.RegistryItem(scope, model, "DELETE", c.delete)
	if err != nil {
		return err
	}

	return nil
}

func (c *Crud) RegistryToManager(m IScenes, scope string, model string) error {
	return c.registryToManager(m, scope, model)
}

func NewCrud(db *qmgo.Database, tableName string, inputStruct any) *Crud {
	return &Crud{
		db:        db,
		tableName: tableName,
		input:     inputStruct,
	}
}
