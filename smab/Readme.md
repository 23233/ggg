### mongo版本 admin管理


#### mab标签可选

* t 代表前端渲染的类型
    * img 作用于表格 会把值作为 img 的src 进行图片显示
        * thumbnail 可描述为缩略图模式
    * textarea 作用于表单 会渲染为textarea输入框
    * markdown 作用于表单 会渲染为markdown编辑器
* fk 外键
    * 例子: fk=User
    * 可选 col=[FieldName] 即指定映射字段信息 eg: fk=EventManage,col=Name 建议只使用id primitive.ObjectID类型

##### 图表接口

* 发送接口不论是否在本站 直接发送 会附带4个参数
    * GET
        * chart_id
        * user_id
        * screen_id
        * chart_type
* 返回的内容必须是一个json 可以全覆盖开始设置的配置文件
    * 内容不能传错 会错误而崩溃 (暂时不处理)

#### 动作

* 发送接口一定是POST请求 会附带4个必备参数
    * POST
        * record 代表选中行的数据(可能会变更请注意时间戳 太长了就不可信了)
        * t 当前时间戳
        * action 动作的名称
        * table 当前表名
        * form? 可选 如果设置了对应的scheme 则会把表单结果传递
            * 如果表单没有一个必填项 若未有任何输入则不会出现此字段信息
* 因为是前端直接发送 会面临跨域问题 暂不考虑进行后端转发 请酌情处理 若要后端转发 请提issue 会安排

#### 任务

* 创建任务需要在代码层完成
* 任务的action form 都采用表单生成器 使用post发送数据
    * POST
        * sm_task_id
        * sm_user_id
        * sm_action_name action的名称
        * built 为内置传递内容 json
        * form? 如果设置了对应的scheme 则会把表单结果传递
            * 只要设置了此字段 即时没有任何必填项 也会包含此字段请求 只是值可能为空

#### 子应用(微前端)

* 使用umi-qiankun [文档](https://umijs.org/zh-CN/plugins/plugin-qiankun)
* 自动向子应用注入 package对象 内有 userToken userInfo userPer 对应当前用户token 当前用户信息 当前用户权限
    * 子应用通过 const package = useModel('@@qiankunStateFromMaster') 即可获取数据

#### 问题

* ~~权限casbin是通过本地的sqlite保存 可能不利于横向拓展 后期若有需求再变更~~
* 登录未做爆破验证 可以选择自己前置rate

#### 提示

* 配置的图表信息参数都依赖于 [charts](https://charts.ant.design/)
* 用户的创建者对用户有管理权 即使账号的权重权限已经超过了创建者但创建者依然有最大的控制权
    * 虽然使用了rbac 但是考虑到后台的逻辑性和特殊性 还是以级别相对控制

#### 记录

* bson标签的flatten主要用于 `mongo-go-struct-to-bson` 这个库