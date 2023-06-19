package ut

import (
	"errors"
	"strings"
	"sync"
)

// 前缀树匹配
// 动态路径使用:号分割 eg: /api/:board 会返回 map[string]string{"board":""}

type TrieNode struct {
	isTail         bool // 是否是尾部
	isDynamic      bool // 是否为动态匹配
	methodsMapper  map[string]any
	next           map[string]*TrieNode
	supportMethods []string
	pathMatch      map[string]string // 路径最终的匹配
}

func (c *TrieNode) GetVal(method string) any {
	if v, ok := c.methodsMapper[method]; ok {
		return v
	}
	return nil
}

func (c *TrieNode) IsValidMethod(method string) bool {
	for _, supportMethod := range c.supportMethods {
		if supportMethod == method {
			return true
		}
	}
	return false
}

// GetMatchParams 获取匹配的参数 :board 但不会校验格式
func (c *TrieNode) GetMatchParams() map[string]string {
	return c.pathMatch
}

type TrieMatch struct {
	mutex  *sync.RWMutex
	root   *TrieNode
	prefix string // 路径前缀 仅记录用
}

func (t *TrieMatch) Prefix() string {
	return t.prefix
}

func (t *TrieMatch) SetPrefix(prefix string) {
	t.prefix = prefix
}

func NewTrieMatch() *TrieMatch {
	return &TrieMatch{
		root:  &TrieNode{next: map[string]*TrieNode{}, supportMethods: make([]string, 0)},
		mutex: new(sync.RWMutex),
	}
}

func (t *TrieMatch) validDynamic(part string) bool {
	return strings.HasPrefix(part, ":")
}

func (t *TrieMatch) parsePattern(pattern string) []string {
	pattern = strings.Trim(pattern, "/")
	parts := strings.Split(pattern, "/")
	for idx, val := range parts {
		parts[idx] = val + "/"
	}
	return parts
}

// Add 新增对应路线与方法 相同路径 不同方法都会存在 相同方法会直接覆盖
func (t *TrieMatch) Add(pattern string, method string, raw any) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	parts := t.parsePattern(pattern)
	cur := t.root
	for _, part := range parts {
		if _, ok := cur.next[part]; !ok {
			newNodes := &TrieNode{next: map[string]*TrieNode{}}
			if t.validDynamic(part) {
				for _, node := range cur.next {
					if node.isDynamic {
						err := errors.New("YOU CANNOT ADD THE SAME DYNAMIC RULE TO THE SAME ROUTE")
						return err
					}
				}
				newNodes.isDynamic = true
			} else {
				newNodes.isDynamic = false
			}
			cur.next[part] = newNodes
		}
		cur = cur.next[part]
	}
	cur.isTail = true
	if cur.methodsMapper == nil {
		cur.methodsMapper = make(map[string]any, 0)
	}
	cur.methodsMapper[method] = raw
	cur.supportMethods = append(cur.supportMethods, method)
	return nil
}

func (t *TrieMatch) get(pattern string) *TrieNode {
	parts := t.parsePattern(pattern)
	cur := t.root
	pathParams := make(map[string]string)
	for _, part := range parts {
		// 下级没有的话 判断下级是否是动态参数
		if _, ok := cur.next[part]; !ok {
			foundDynamic := false
			// 遍历下级是否为动态
			for keyPart, node := range cur.next {
				if node.isDynamic {
					foundDynamic = true
					partPath := strings.TrimLeft(keyPart, ":")
					partPath = strings.TrimRight(partPath, "/")
					pathParams[partPath] = strings.TrimRight(part, "/")
					part = keyPart
				}
			}

			if !foundDynamic {
				return nil
			}
		}
		cur = cur.next[part]
	}

	cur.pathMatch = pathParams

	return cur
}

// Get 获取 支持正则
func (t *TrieMatch) Get(pattern string) *TrieNode {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.get(pattern)
}

func (t *TrieMatch) pruneGet(pattern string) *TrieNode {
	parts := t.parsePattern(pattern)
	cur := t.root
	for _, part := range parts {
		// 如果下级没有 则直接返回 这时候不是匹配 所以不需要正则
		if _, ok := cur.next[part]; !ok {
			return nil
		}
		cur = cur.next[part]
	}
	return cur
}

// PruneGet 仅匹配纯粹输入 不会解析动态参数 eg: /a/:board
func (t *TrieMatch) PruneGet(pattern string) *TrieNode {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.pruneGet(pattern)
}

// RemoveMethod 删除匹配项对应的方法 eg: /a/b/c [GET,POST] --> /a/b/c [GET]
// 但是如果路径下仅此一个方法 则会执行删除路径 eg: /a/b/c [GET] - GET --> /a/b
func (t *TrieMatch) RemoveMethod(pattern, method string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	cur := t.pruneGet(pattern)
	if cur != nil {
		if cur.IsValidMethod(method) {
			// 如果仅此一个方法 则执行删除路径
			if len(cur.supportMethods) == 1 {
				t.removeTrie(pattern)
				return
			}
			methods := make([]string, 0, len(cur.supportMethods)-1)
			for _, supportMethod := range cur.supportMethods {
				if supportMethod != method {
					methods = append(methods, supportMethod)
				}
			}
			cur.supportMethods = methods
		}
	}

	return
}

func (t *TrieMatch) UpdateRaw(pattern, method string, raw any) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	cur := t.pruneGet(pattern)
	if cur != nil {
		cur.methodsMapper[method] = raw
	}
}

func (t *TrieMatch) removeTrie(pattern string) {
	parts := t.parsePattern(pattern)
	parent := new(TrieNode)
	cur := t.root
	for _, part := range parts {
		// 如果下级没有 则直接返回 这时候不是匹配 所以不需要正则
		if _, ok := cur.next[part]; !ok {
			return
		}
		*parent = *cur
		cur = cur.next[part]
	}
	// 找到最后一条路径 然后删除
	last := parts[len(parts)-1]
	delete(parent.next, last)
}

// Remove 仅删除最后一个匹配项 eg : /a/b/c -> /a/b
func (t *TrieMatch) Remove(pattern string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.removeTrie(pattern)
}

// Match 支持动态参数匹配
func (t *TrieMatch) Match(pattern string, method string) (*TrieNode, error) {
	node := t.Get(pattern)
	// 获取成功
	if node != nil {
		if node.IsValidMethod(method) {
			return node, nil
		} else {
			return nil, errors.New("检测到未受支持的路由方法")
		}
	}
	return nil, errors.New("未获取到匹配的路由信息")
}
