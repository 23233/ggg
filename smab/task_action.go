package smab

// 表单数据在线设计 https://x-render.gitee.io/tools/generator/playground

// PassOrNotReasonAction 通过或者不通过的action 不通过需要输入理由 built 为内置json str
func PassOrNotReasonAction(passUrl string, notPassUrl string, built string) []ActionItem {
	return []ActionItem{PassAction(passUrl, built), RejectAction(notPassUrl, built)}
}

// PassOrRejectAction 通过或拒绝action 共用一个uri
func PassOrRejectAction(uri string, built string) []ActionItem {
	return []ActionItem{PassAction(uri, built), RejectAction(uri, built)}
}

func PassAction(url string, built string) ActionItem {
	return CreateAction("通过", url, "", built)
}

func RejectAction(url string, built string) ActionItem {
	return CreateAction("不通过", url, `{ "type": "object", "properties": { "reason": { "title": "请输入拒绝理由", "type": "string", "format": "textarea", "props": { "autoSize": true }, "displayType": "row", "required": true, "labelWidth": 150, "maxLength": 200 } }, "labelWidth": 120, "displayType": "row" }`, built)
}

// CreateAction 创建一个action
func CreateAction(name string, toUrl string, scheme string, built string) ActionItem {
	return ActionItem{
		Name:   name,
		ReqUri: toUrl,
		Built:  built,
		Scheme: scheme,
	}
}
