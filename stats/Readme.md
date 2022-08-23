# 使用redis进行各类统计信息的方法汇总

### PV UV 统计

* 使用 redis `HyperLogLog` 结构实现
* 支持任何事件级统计 例如
    * 页面PV UV 或 按钮点击次数
* 默认自动以当前时间作为 `key` 格式为 `[prefix]:[event]:[date]`  
  * 也可以主动变更为最后的日期为任意字符串 格式为 `[prefix]:[event]:[string]`
* 可汇聚任意时间段内进行统计
* 以 `must` 开头的函数 均没有返回值 