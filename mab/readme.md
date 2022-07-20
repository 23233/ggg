#### ab

* page 控制页码 page_size 控制条数
    * 最大均为100 100页 100条
* _last mid 大页码通用
* _o(asc) _od(desc) 排序 _o=like _od=`字段名`
* _s搜索 __在左右则为模糊 _s=__赵日天
* [字段名] 进行过滤 id=1 最长64位请注意 and关系 eg name=赵日天
* _o_[字段名] 进行过滤 _o_id=2 最长64位 or关系 eg _o_name=赵日天

* _g $geoNear 根据此点返回由近到远的列表 _g=`lng,lat`
    * 一旦出现此参数则会自动进入Aggregate执行模式且$geoNear排名第一位
    * 官方文档[点这里](https://docs.mongodb.com/manual/reference/operator/aggregation/geoNear/)
    * 默认 distanceField 为 _distance
    * 使用geojson方式 所以返回的距离为米
    * spherical 为true 是球面计算方式
    * maxDistance 取参数`_gmax` 单位为米 可不传
    * minDistance 取参数`_gmin` 单位为米 可不传
    * key 默认为location 暂不支持修改

* _gmax 必须与_g参数同时出现才有意义 为最大距离 单位米
* _gmin 必须与_g参数同时出现才有意义 为最小距离 单位米

#### 别名配置

* 表别名 struct方法
    * Alias() string 或者 SpAlias() string 均可配置表别名 格式为 `_组名-1_表名-1` 也可以为仅返回表名 会进入默认的 `未命名` 组
        * 组名和表名 后面如果是 `-1` 则是排序的数组 会自动去除 数字越大 排名优先
* 字段别名 comment 标签

#### 说明

* 全面拥抱json 新增修改均使用json上传
* 约定大于配置

```shell
struct定义的名称
  * Id 为主键
  * UpdateAt 为更新时间 每次更新自动赋值
  * CreateAt 为创建时间 需要自己设置qmgo的插入事件或自行设置时间
  * DeleteAt 为删除时间 一般不需要设置 除非使用了悲观锁version 需要自行设置

外键
  * 配置项PK函数 需要返回数组的lookup stage阶段所需的bson  

参数操作符
  * 通过_[op] 或 _[op]_ 可进行操作符定义 eg: params: address__position_eq_="赵日天" 会转换成 {"address.position":{"$eq":"赵日天"}}
  * 支持的op有 `eq`, `gt`, `gte`, `lt`, `lte`, `ne` , `in`, `nin`,`exists`,`null`
  * `exists` 不是内容 而是指字段是否存在 例如 `name_exists_`:`true`|`false` 则表示`name`字段是否存在
  * `null` 表示内容是否存在 eg: `name_null_`:`true`|`false` 则表示判断`name`字段内容是否存在

```

* 请一定要设置bson标签 若不设置则自动小写的蛇形
* 修改成功后返回的是成功变更的字段序列
* private context上下文传递的时候类型一定要同struct定义的一致
* 仅新增是通过struct方式 所以可以触发qmgo的事件
* 修改更新时会自动发现updated 自动赋值最新时间 使用的是 time.Now().Local()
* 关于时间时区问题 默认UTC RFC3339Nano
* 大量使用了反射 主要满足前期快速开发的业务需求 上一定量之后可以考虑深入优化
* 各种mustFilters 虽然用的map string 但是只会考虑key value会被忽略 暂时..
* 对于要进行比较的time 必须是utc时间!!!

#### 限制

* ~~暂未开放cache~~
* 暂时使用本地化缓存 有需要的再加入集群缓存

