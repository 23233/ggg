## 后端
* 方法参数交互式后端定义
* 支持同一个struct存储
* 但是不同的行为支持设置不同的struct
* 每个行为都有一个默认值
* 一个入口方便加密
```json lines
{
  "module": {
    "scope": "", // 作用域 例如说 miniapp admin
    "model": "", // 对应的模型 tasks
    "scene": "", // 对应的场景选填 某些action必填
    "action": "", // 对应的操作 getAllModel
  },
  "query": {}, // 这里是map[string]string 也顺理成章了
  "body": "", //这里是json字符串 那么就顺理成章了
  "comm":{} // 这里放置一些通用信息 版本号之类的 map即可
}
```

#### scene列表
- table
- add
- edit
等等可以任意
 
