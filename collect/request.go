package collect

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"regexp"
	"time"
)

type Property struct {
	Name     string        `json:"name"`
	Url      string        `json:"url"`
	Cookie   string        `json:"cookie"`
	WaitTime time.Duration `json:"wait_time"`
	Reload   bool          `json:"reload"`
	MaxDepth int64         `json:"max_depth"`
}

// Task 任务实例
type Task struct {
	Property
	RootReq *Request
	Fetcher Fetcher
	Rule    RuleTree
}

type Context struct {
	Body []byte
	Req  *Request
}

type Request struct {
	Task     *Task
	Url      string
	Method   string
	Depth    int64
	Priority int64
	RuleName string
}

type ParseResult struct {
	Requests []*Request
	Items    []interface{}
}

// Check 校验最大深度
func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return errors.New("max depth limit reached")
	}
	return nil
}

// Unique 生成唯一标识，用于去重
func (r *Request) Unique() string {
	block := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(block[:])
}

func (c *Context) ParseJSReg(name string, reg string) ParseResult {
	re := regexp.MustCompile(reg)

	matches := re.FindAllSubmatch(c.Body, -1)
	result := ParseResult{}

	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(result.Requests, &Request{
			Method:   "GET",
			Task:     c.Req.Task,
			Url:      u,
			Depth:    c.Req.Depth + 1,
			RuleName: name,
		})
	}
	return result
}

func (c *Context) OutputJS(reg string) ParseResult {
	re := regexp.MustCompile(reg)
	ok := re.Match(c.Body)
	if !ok {
		return ParseResult{
			Items: []interface{}{},
		}
	}

	result := ParseResult{
		Items: []interface{}{c.Req.Url},
	}
	return result
}
