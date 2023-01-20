package collect

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"time"
)

type Task struct {
	Url      string
	Cookie   string
	WaitTime time.Duration
	MaxDepth int
	RootReq  *Request
	Fetcher  Fetcher
	Reload   bool
}

type Request struct {
	Task      *Task
	Url       string
	Method    string
	Depth     int
	Priority  int
	ParseFunc func([]byte, *Request) ParseResult
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
