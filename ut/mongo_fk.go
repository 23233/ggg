package ut

import "go.mongodb.org/mongo-driver/bson"

func MUserFk(local, remark string) bson.D {
	return MBuilderFk("user", local, "_id", remark)
}

func MUserInfoFk(localField ...string) []bson.D {
	var local = "user_id"
	var remark = "userInfo"
	switch len(localField) {
	case 1:
		local = localField[0]
		break
	case 2:
		local = localField[0]
		remark = localField[1]
		break
	}
	return MBuildFkUnwind("user", local, "_id", remark)
}

// MBuilderFk 构建通用外键
func MBuilderFk(collection, localField, remoteField, remark string) bson.D {
	look := bson.D{{"$lookup", bson.D{
		{"from", collection},
		{"localField", localField},
		{"foreignField", remoteField},
		{"as", remark},
	}}}
	return look
}

// MBuildFkUnwind 构建通用解构外键
func MBuildFkUnwind(collection, localField, remoteField, remark string) []bson.D {
	look := MBuilderFk(collection, localField, remoteField, remark)
	unwind := bson.D{{
		Key:   "$unwind",
		Value: "$" + remark}}
	return []bson.D{look, unwind}
}

// MBuildFkUnwindOfEmptyReturn preserveNullAndEmptyArrays 默认为false 不存在时则不返回
// 文档 https://docs.mongodb.com/manual/reference/operator/aggregation/unwind/
func MBuildFkUnwindOfEmptyReturn(collection, localField, remoteField, remark string) []bson.D {
	look := MBuilderFk(collection, localField, remoteField, remark)
	unwind := bson.D{{
		Key: "$unwind",
		Value: bson.M{
			"path":                       "$" + remark,
			"preserveNullAndEmptyArrays": true,
		},
	}}
	return []bson.D{look, unwind}
}
