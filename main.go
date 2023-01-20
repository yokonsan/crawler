package main

import (
	"fmt"
	"github.com/yokonsan/crawler/collect"
	"github.com/yokonsan/crawler/engine"
	"github.com/yokonsan/crawler/log"
	"github.com/yokonsan/crawler/parse/doubangroup"
	"go.uber.org/zap/zapcore"
	"time"
)

func main() {
	// log
	plugin := log.NewStdoutPlugin(zapcore.DebugLevel)
	logger := log.NewLogger(plugin)
	logger.Debug("log init end")

	// proxy
	//proxyURLs := []string{"http://127.0.0.1:7890"}
	//p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	//if err != nil {
	//	logger.Error("RoundRobinProxySwitcher failed")
	//}
	cookies := ""
	var seeds = make([]*collect.Task, 0, 1000)
	for i := 0; i <= 100; i += 25 {
		str := fmt.Sprintf("https://www.douban.com/group/zf365/discussion?start=%d", i)
		seeds = append(seeds, &collect.Task{
			Url:      str,
			WaitTime: time.Second * 1,
			Cookie:   cookies,
			RootReq: &collect.Request{
				ParseFunc: doubangroup.ParseURL,
			},
		})
	}

	var f collect.Fetcher = collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		//Proxy:   p,
		Logger: logger,
	}

	s := engine.NewEngine(
		engine.WithWorkCount(4),
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)
	s.Run()
}
