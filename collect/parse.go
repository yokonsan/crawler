package collect

// RuleTree 采集规则树
type RuleTree struct {
	Root  func() ([]*Request, error) // 根结点（执行入口）
	Trunk map[string]*Rule           // 规则哈希
}

// Rule 采集规则节点
type Rule struct {
	ParseFunc func(*Context) (ParseResult, error) //内容解析函数
}
