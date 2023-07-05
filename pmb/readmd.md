## 意义

基于 json schema 的增删改查管理面板
通过struct生成json schema 并且支持自定义action (表级和行级)

## 表

* 用户表
* 权限表

### 本地表

* schema表
* 模型表
    * 传入一个struct以及配置文件 均以json tag为key
        * 禁用那些方法
        * 中文组名
        * 中文别名
        * 优先级
        * 可注入filter
        * 可设置必传filter
        * 可设置必传body
        * 可注入body
        * 返回一个struct 可以动态变更而这个struct写入一个structs队列中
        * 以struct的名称作为key和表名仅支持英文
        * 保留原始raw传入的struct以泛型的形式
        * 保留解析后的struct结构 可以后续变更

### 涉及

* mongodb
* redis
* gcache
* casbin
* iris
* schema

## 行为

* 登录注册
    * 手机号
        * 发送验证码
            * 必须过滑块验证
    * 用户名密码
    * 判断登录环境是否允许登录
* 滑块验证码
    * 获取
    * 验证

* 获取模型
    * 增
    * 删
    * 改
    * 查
    * 自定义action
        * 表级
        * 行级
        * form
            * schema
    * 获取模型定义

## 前端

* 基于json schema的表格 表单 