package ut

import (
	"fmt"
	"gorm.io/gorm"
	"strings"
)

type GormQueryResult[T any] struct {
	Count int64 `json:"count"`
	Datas []*T  `json:"datas"`
}

// Helper function to apply conditions
func applyCondition(db *gorm.DB, key string, op string, value any, or bool) (*gorm.DB, error) {
	var condition string
	switch op {
	case "eq":
		condition = fmt.Sprintf("%s = ?", key)
	case "gt":
		condition = fmt.Sprintf("%s > ?", key)
	case "gte":
		condition = fmt.Sprintf("%s >= ?", key)
	case "lt":
		condition = fmt.Sprintf("%s < ?", key)
	case "lte":
		condition = fmt.Sprintf("%s <= ?", key)
	case "ne":
		condition = fmt.Sprintf("%s != ?", key)
	case "in":
		condition = fmt.Sprintf("%s IN ?", key)
	case "nin":
		condition = fmt.Sprintf("%s NOT IN ?", key)
	case OpRegex:
		likePattern := "%" + strings.ReplaceAll(value.(string), ".*", "%") + "%"
		condition = fmt.Sprintf("%s LIKE ?", key)
		value = likePattern
	default:
		return nil, fmt.Errorf("unsupported operator: %s", op)
	}

	if or {
		return db.Or(condition, value), nil
	}
	return db.Where(condition, value), nil
}

// BuildGormQuery 使用query *QueryFull 构建出对应的查询
func BuildGormQuery[T any](query *QueryFull, db *gorm.DB) (*gorm.DB, error) {
	q := db.Model(new(T))
	if query == nil {
		query = &QueryFull{}
	}
	if query.QueryParse == nil {
		query.QueryParse = &QueryParse{}
	}
	if query.BaseQuery == nil {
		query.BaseQuery = &BaseQuery{}
	}
	if query.BaseQuery.BasePage == nil {
		// 默认必须有page 和size 不然就获取全部了
		query.BaseQuery.BasePage = &BasePage{
			Page:     1,
			PageSize: 10,
		}
	}
	if query.BaseQuery.BaseSort == nil {
		query.BaseQuery.BaseSort = &BaseSort{}
	}

	// Step 1: Apply AND conditions
	for _, and := range query.And {
		var err error
		q, err = applyCondition(q, and.Key, and.Op, and.Value, false)
		if err != nil {
			return nil, err
		}
	}

	// Step 2: Apply OR conditions
	for _, or := range query.Or {
		var err error
		q, err = applyCondition(q, or.Key, or.Op, or.Value, true)
		if err != nil {
			return nil, err
		}
	}

	// Step 3: Apply sorting
	for _, asc := range query.SortAsc {
		q = q.Order(fmt.Sprintf("%s ASC", asc))
	}
	for _, desc := range query.SortDesc {
		q = q.Order(fmt.Sprintf("%s DESC", desc))
	}

	// Step 4: Handle pagination
	if query.Page > 0 && query.PageSize > 0 {
		offset := (query.Page - 1) * query.PageSize
		q = q.Offset(int(offset)).Limit(int(query.PageSize))
	}
	return q, nil
}

// RunGormQuery 使用query 解析出gorm的查询结果
func RunGormQuery[T any](query *QueryFull, db *gorm.DB) (*GormQueryResult[T], error) {
	q, err := BuildGormQuery[T](query, db)
	if err != nil {
		return nil, err
	}
	// Step 5: Execute query
	var result GormQueryResult[T]
	if query.GetCount {
		var count int64
		if err := q.Count(&count).Error; err != nil {
			return nil, err
		}
		result.Count = count
	}

	var datas []*T
	if err := q.Find(&datas).Error; err != nil {
		return nil, err
	}
	result.Datas = datas

	return &result, nil
}
