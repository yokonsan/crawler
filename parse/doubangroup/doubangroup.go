package doubangroup

import (
	"fmt"
	"github.com/yokonsan/crawler/collect"
	"regexp"
	"time"
)

const urlListRe = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`
const ContentRe = `<div class="topic-content">[\s\S]*?铁心桥[\s\S]*?<div`

var DoubangroupTask = &collect.Task{
	Property: collect.Property{
		Name:     "find_douban_sun_room",
		WaitTime: 1 * time.Second,
		MaxDepth: 5,
		Cookie:   "",
	},
	Rule: collect.RuleTree{
		Root: func() ([]*collect.Request, error) {
			var roots []*collect.Request
			for i := 0; i <= 25; i += 25 {
				str := fmt.Sprintf("https://www.douban.com/group/zf365/discussion?start=%d", i)
				roots = append(roots, &collect.Request{
					Url:      str,
					Priority: 1,
					Method:   "GET",
					RuleName: "解析网站URL",
				})
			}
			return roots, nil
		},
		Trunk: map[string]*collect.Rule{
			"解析网站URL": &collect.Rule{ParseURL},
			"解析阳台房":   &collect.Rule{GetSunRoom},
		},
	},
}

func ParseURL(ctx *collect.Context) (collect.ParseResult, error) {
	re := regexp.MustCompile(urlListRe)

	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(result.Requests, &collect.Request{
			Url:      u,
			Method:   "GET",
			Task:     ctx.Req.Task,
			Depth:    ctx.Req.Depth + 1,
			RuleName: "解析阳台房",
		})
	}
	return result, nil
}

func GetSunRoom(ctx *collect.Context) (collect.ParseResult, error) {
	re := regexp.MustCompile(ContentRe)

	ok := re.Match(ctx.Body)
	if !ok {
		return collect.ParseResult{
			Items: []interface{}{},
		}, nil
	}

	result := collect.ParseResult{
		Items: []interface{}{ctx.Req.Url},
	}
	return result, nil
}
