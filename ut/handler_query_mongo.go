package ut

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func queryToMatch(query *QueryFull) bson.D {
	// bson.D[bson.E{"$and":bson.A[bson.D{bson.E,bson.E}]},{"$or": bson.A{ bson.D{{}}, bson.D{{}}}]
	r := bson.D{}

	andItem := bson.D{}
	for _, value := range query.And {
		if len(value.Op) < 1 {
			andItem = append(andItem, bson.E{
				Key:   value.Key,
				Value: value.Value,
			})
			continue
		}
		andItem = append(andItem, bson.E{
			Key: value.Key,
			Value: bson.M{
				"$" + value.Op: value.Value,
			},
		})
	}
	if len(andItem) > 0 {
		// bson.A[bson.D{bson.E,bson.E}]
		r = append(r, bson.E{
			Key:   "$and",
			Value: bson.A{andItem},
		})
	}
	// {"$or": bson.A{ bson.D{{}}, bson.D{{}}}
	orItem := bson.A{}
	for _, value := range query.Or {
		if len(value.Op) < 1 {
			orItem = append(orItem, bson.E{
				Key:   value.Key,
				Value: value.Value,
			})
			continue
		}
		// 正则
		var be bson.E
		if value.Op == OpRegex {
			be = bson.E{
				Key: value.Key,
				Value: bson.D{
					{"$" + OpRegex, primitive.Regex{Pattern: value.Value.(string), Options: "i"}},
				},
			}
		} else {
			be = bson.E{
				Key: value.Key,
				Value: bson.M{
					"$" + value.Op: value.Value,
				},
			}
		}
		orItem = append(orItem, bson.D{be})
	}
	if len(orItem) > 0 {
		r = append(r, bson.E{
			Key:   "$or",
			Value: orItem,
		})
	}

	return r
}

func parsePk(parse *QueryFull) []bson.D {
	var d = make([]bson.D, 0)
	for _, p := range parse.Pks {
		var b = make([]bson.D, 0)
		if p.EmptyReturn {
			b = MBuildFkUnwindOfEmptyReturn(p.RemoteModelId, p.LocalKey, p.RemoteKey, p.Alias)
		} else {
			b = append(b, MBuilderFk(p.RemoteModelId, p.LocalKey, p.RemoteKey, p.Alias))
		}
		d = append(d, b...)
	}
	return d
}

func fkQuery(matchQuery bson.D, parse *QueryFull) []bson.D {
	// 这里顺序特别重要 一定不能随意变更顺序 geo存在则geo必须在前面 不存在则match在前 lookup中间
	steps := make([]bson.D, 0)

	// https://www.mongodb.com/docs/v4.4/reference/operator/aggregation/geoNear/
	if parse.Geos != nil {
		var geo = parse.Geos
		var near = bson.M{
			"near": bson.M{
				"type":        "Point",
				"coordinates": []float64{geo.Lng, geo.Lat},
			},
			"key":           geo.Field,
			"distanceField": geo.ToField,
			"spherical":     true,
		}
		if geo.GeoMax >= 1 {
			near["maxDistance"] = geo.GeoMax
		}
		if geo.GeoMin >= 1 {
			near["minDistance"] = geo.GeoMin
		}
		if len(matchQuery) > 0 {
			near["matchQuery"] = matchQuery
		}
		geoNear := bson.D{{"$geoNear", near}}
		steps = append(steps, geoNear)
	} else {

		if len(matchQuery) > 0 {
			steps = append(steps, bson.D{{"$match", matchQuery}})
		}

	}

	// 外键
	if len(parse.Pks) >= 1 {
		pks := parsePk(parse)
		if len(pks) > 0 {
			steps = append(steps, pks...)
		}
	}

	// 解析出各种过滤 sort limit skip
	filters := make([]bson.D, 0)

	if parse.BaseQuery != nil {
		// 解析出sort 顺序在limit skip之前
		sort := bson.D{}
		if len(parse.SortDesc) > 0 {
			for _, s := range parse.SortDesc {
				sort = append(sort, bson.E{Key: s, Value: -1})
			}
		}
		if len(parse.SortAsc) > 0 {
			for _, s := range parse.SortAsc {
				sort = append(sort, bson.E{Key: s, Value: 1})
			}
		}

		if len(sort) > 0 {
			filters = append(filters, bson.D{{"$sort", sort}})
		}
		skip := (parse.Page - 1) * parse.PageSize
		if skip > 0 {
			filters = append(filters, bson.D{{"$skip", skip}})
		}
		if parse.PageSize > 0 {
			filters = append(filters, bson.D{{"$limit", parse.PageSize}})
		}

	}

	if parse.GetCount {
		steps = append(steps, bson.D{
			{"$facet",
				bson.D{
					{"meta",
						bson.A{
							bson.D{{"$count", "count"}},
						},
					},
					{"data", filters},
				},
			},
		})

		steps = append(steps, bson.D{{"$unwind", bson.D{{"path", "$meta"}}}})

	} else {
		steps = append(steps, filters...)
	}

	return steps
}

func QueryToMongoPipeline(query *QueryFull) []bson.D {
	matchQuery := queryToMatch(query)
	pipeline := fkQuery(matchQuery, query)
	return pipeline
}

type MongoFacetResult struct {
	Count int64 `json:"count,omitempty"`
	Data  any   `json:"data"`
}
