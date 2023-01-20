package doubangroup

import (
	"github.com/yokonsan/crawler/collect"
	"regexp"
)

const cityListRe = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`

func ParseURL(contents []byte, req *collect.Request) collect.ParseResult {
	re := regexp.MustCompile(cityListRe)

	matches := re.FindAllSubmatch(contents, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(result.Requests, &collect.Request{
			Url:   u,
			Task:  req.Task,
			Depth: req.Depth,
			ParseFunc: func(c []byte, req *collect.Request) collect.ParseResult {
				return GetContent(c, u)
			},
		})
	}
	return result
}

const ContentRe = `<div class="topic-content">[\s\S]*?铁心桥[\s\S]*?<div`

func GetContent(contents []byte, url string) collect.ParseResult {
	re := regexp.MustCompile(ContentRe)

	ok := re.Match(contents)
	if !ok {
		return collect.ParseResult{
			Items: []interface{}{},
		}
	}

	result := collect.ParseResult{
		Items: []interface{}{url},
	}
	return result
}
