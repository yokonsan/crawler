package main

import (
	"github.com/yokonsan/crawler/collect"
	"github.com/yokonsan/crawler/engine"
	"github.com/yokonsan/crawler/log"
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
	var f collect.Fetcher = collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		//Proxy:   p,
		Logger: logger,
	}
	var seeds = make([]*collect.Task, 0, 1000)
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "js_find_douban_sun_room",
		},
		Fetcher: f,
	})

	s := engine.NewEngine(
		engine.WithWorkCount(5),
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)
	s.Run()
}
