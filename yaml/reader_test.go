package yaml

import (
	"bytes"
	"encoding/json"
	"log"
	"testing"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func TestSpileToken(*testing.T) {
	strs := []string{"name : wendal",
		"skill : [c,c++,java,lua]",
		"study:   { ppp:123 , vv : 23 }",
		"study:   { ppp:        123 , vv : 23 }",
		"study :   { ppp : 123 , vv : 23 }   ",
		"a:b",
		"a:[1,2,3,4,5,6,7,8]"}

	for _, str := range strs {
		tokens := splitToken(str)
		data, _ := json.Marshal(tokens)
		log.Println(str, string(data))
	}
}

func TestAsMapKeyValue(*testing.T) {
	strs := []string{"name : wendal",
		"skill : [c,c++,java,lua]",
		"study:   { ppp:123 , vv : 23 }",
		"study:   { ppp:        123 , vv : 23 }",
		"study :   { ppp : 123 , vv : 23 }   ",
		"a:b",
		"a:[1,2,3,4,5,6,7,8]",
		"abc : {abc:abc, ccc:zzz,sdfsadfasf:asfasfasfasfasfasfasfasdf}"}
	yaml := &yamlReader{}
	for _, str := range strs {
		key, value, err := yaml.asMapKeyValue(str)
		log.Println(key, value, err)
	}
}

func TestReadList(*testing.T) {
	strs := []string{`
- name
- age 
`, `
name : wendal
age  : 27
skill : [c, c++, java]
study : 
  gdut : 2004
  jnu  : 2008
`, `
tags :
- java
- qq
- golang
catalog :
- Jobs
`, `
---
comments: true
date: 2012-01-06 22:20:11
layout: post
slug: nutdao%e9%85%8d%e7%bd%ae%e5%a4%9a%e6%95%b0%e6%8d%ae%e6%ba%90
title: NutDao配置多数据源
permalink: '/356.html'
wordpress_id: 356
categories:
- Java
tags:
- el
- io
- Nutz
- 连接池
- 配置
---`, `
layout : disqus
disqus :
  short_name : wendalblog
livefyre :
  site_id : 123
intensedebate :
  account : 123abc
facebook :
  appid : 123
  num_posts: 5
  width: 580
  colorscheme: light
`, `linenums : false`, `
javascripts:
  [processing-1.4.1.min.js]

layout: processing

`, `
javascripts:
    [processing-1.4.1.min.js]

layout: processing

`}

	for _, str := range strs {
		log.Println("!!>>> ", str)
		obj, err := Read(bytes.NewBufferString(str))
		log.Println("*********", str, obj, err, "*******")
	}
}
