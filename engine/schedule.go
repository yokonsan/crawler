package engine

import (
	"github.com/robertkrimen/otto"
	"github.com/yokonsan/crawler/collect"
	"github.com/yokonsan/crawler/parse/doubangroup"
	"go.uber.org/zap"
	"sync"
)

type Crawler struct {
	out         chan collect.ParseResult
	Visited     map[string]bool
	VisitedLock sync.Mutex

	failures    map[string]*collect.Request
	failureLock sync.Mutex
	options
}

type Scheduler interface {
	Schedule()
	Push(...*collect.Request)
	Pull() *collect.Request
}

type Schedule struct {
	requestCh   chan *collect.Request
	workerCh    chan *collect.Request
	priReqQueue []*collect.Request
	reqQueue    []*collect.Request
	Logger      *zap.Logger
}

type CrawlerStore struct {
	list []*collect.Task
	hash map[string]*collect.Task
}

// Store 全局爬虫任务实例
var Store = &CrawlerStore{
	list: []*collect.Task{},
	hash: map[string]*collect.Task{},
}

func init() {
	Store.Add(doubangroup.DoubangroupTask)
	Store.AddJSTask(doubangroup.DoubangroupJSTask)
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	c := &Crawler{}
	c.Visited = make(map[string]bool, 100)
	c.out = make(chan collect.ParseResult)
	c.failures = make(map[string]*collect.Request)
	c.options = options
	return c
}

func NewSchedule() *Schedule {
	return &Schedule{
		requestCh: make(chan *collect.Request),
		workerCh:  make(chan *collect.Request),
	}
}

// AddJsReqs 用于动态规则添加请求
func AddJsReqs(jreqs []map[string]interface{}) []*collect.Request {
	reqs := make([]*collect.Request, 0)
	for _, jreq := range jreqs {
		req := &collect.Request{}
		u, ok := jreq["Url"].(string)
		if !ok {
			return nil
		}

		req.Url = u
		req.RuleName, _ = jreq["RuleName"].(string)
		req.Method, _ = jreq["Method"].(string)
		req.Priority, _ = jreq["Priority"].(int64)
		reqs = append(reqs, req)
	}

	return reqs
}

func AddJsReq(jreq map[string]interface{}) []*collect.Request {
	reqs := make([]*collect.Request, 0)
	req := &collect.Request{}
	u, ok := jreq["Url"].(string)
	if !ok {
		return nil
	}

	req.Url = u
	req.RuleName, _ = jreq["RuleName"].(string)
	req.Method, _ = jreq["Method"].(string)
	req.Priority, _ = jreq["Priority"].(int64)
	reqs = append(reqs, req)
	return reqs
}

func (c *CrawlerStore) AddJSTask(m *collect.TaskModule) {
	task := &collect.Task{
		Property: m.Property,
	}

	task.Rule.Root = func() ([]*collect.Request, error) {
		vm := otto.New()
		vm.Set("AddJsReq", AddJsReqs)
		v, err := vm.Eval(m.Root)
		if err != nil {
			return nil, err
		}
		e, err := v.Export()
		if err != nil {
			return nil, err
		}

		return e.([]*collect.Request), nil
	}

	for _, r := range m.Rules {
		parseFunc := func(parse string) func(ctx *collect.Context) (collect.ParseResult, error) {
			return func(ctx *collect.Context) (collect.ParseResult, error) {
				vm := otto.New()
				vm.Set("ctx", ctx)
				v, err := vm.Eval(parse)
				if err != nil {
					return collect.ParseResult{}, err
				}

				e, err := v.Export()
				if err != nil {
					return collect.ParseResult{}, err
				}
				if e == nil {
					return collect.ParseResult{}, err
				}
				return e.(collect.ParseResult), err
			}
		}(r.ParseFunc)

		if task.Rule.Trunk == nil {
			task.Rule.Trunk = make(map[string]*collect.Rule, 0)
		}
		task.Rule.Trunk[r.Name] = &collect.Rule{
			ParseFunc: parseFunc,
		}
	}

	c.hash[task.Name] = task
	c.list = append(c.list, task)
}

func (c *CrawlerStore) Add(task *collect.Task) {
	c.hash[task.Name] = task
	c.list = append(c.list, task)
}

func (c *Crawler) SetFailure(req *collect.Request) {
	if !req.Task.Reload {
		c.VisitedLock.Lock()
		unique := req.Unique()
		delete(c.Visited, unique)
		c.VisitedLock.Unlock()
	}

	c.failureLock.Lock()
	defer c.failureLock.Unlock()
	if _, ok := c.failures[req.Unique()]; !ok {
		// 首次失败，重试1次
		c.failures[req.Unique()] = req
		c.scheduler.Push(req)
	}
	// todo：失败2次，加载到队列

}

func (c *Crawler) HasVisited(r *collect.Request) bool {
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()

	return c.Visited[r.Unique()]
}

func (c *Crawler) StoreVisited(reqs ...*collect.Request) {
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()

	for _, req := range reqs {
		c.Visited[req.Unique()] = true
	}
}

func (c *Crawler) Run() {
	go c.Schedule()
	for i := 0; i < c.WorkCount; i++ {
		go c.CreateWork()
	}
	c.HandleResult()
}

func (c *Crawler) Schedule() {
	var reqs []*collect.Request
	for _, seed := range c.Seeds {
		task := Store.hash[seed.Name]
		task.Fetcher = seed.Fetcher
		// 获取初始化任务
		rootreqs, _ := task.Rule.Root()
		for _, req := range rootreqs {
			req.Task = task
		}
		reqs = append(reqs, rootreqs...)
	}

	go c.scheduler.Schedule()
	go c.scheduler.Push(reqs...)
}

func (c *Crawler) CreateWork() {
	for {
		req := c.scheduler.Pull()
		if err := req.Check(); err != nil {
			c.Logger.Error("check failed ", zap.Error(err))
			continue
		}
		if !req.Task.Reload && c.HasVisited(req) {
			c.Logger.Debug("request has visited ", zap.String("url: ", req.Url))
			continue
		}
		c.StoreVisited(req)

		body, err := req.Task.Fetcher.Get(req)
		if err != nil {
			c.Logger.Error("can't fetch ", zap.Error(err))
			c.SetFailure(req)
			continue
		}

		// 获取当前任务对应的规则
		rule := req.Task.Rule.Trunk[req.RuleName]
		result, _ := rule.ParseFunc(&collect.Context{
			Body: body,
			Req:  req,
		})
		// 新任务加入队列
		if len(result.Requests) > 0 {
			go c.scheduler.Push(result.Requests...)
		}
		c.out <- result
	}
}

func (c *Crawler) HandleResult() {
	for {
		select {
		case result := <-c.out:
			for _, item := range result.Items {
				// todo: store
				c.Logger.Sugar().Info("get result: ", item)
			}
		}
	}
}

func (s *Schedule) Push(reqs ...*collect.Request) {
	for _, req := range reqs {
		s.requestCh <- req
	}
}

func (s *Schedule) Pull() *collect.Request {
	return <-s.workerCh
}

func (s *Schedule) Schedule() {
	var req *collect.Request
	var ch chan *collect.Request
	for {
		if req == nil && len(s.priReqQueue) > 0 {
			req = s.priReqQueue[0]
			s.priReqQueue = s.priReqQueue[1:]
			ch = s.workerCh
		}
		if req == nil && len(s.reqQueue) > 0 {
			req = s.reqQueue[0]
			s.reqQueue = s.reqQueue[1:]
			ch = s.workerCh
		}
		select {
		case r := <-s.requestCh:
			if r.Priority > 0 {
				s.priReqQueue = append(s.priReqQueue, r)
			} else {
				s.reqQueue = append(s.reqQueue, r)
			}
		case ch <- req:
			req = nil
			ch = nil
		}
	}
}
