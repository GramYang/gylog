package main

import (
	"fmt"
	g "gylog"
	"time"
)

func main() {
	//基本测试
	//t1()
	//rotate
	//t2()
}

func t1() {
	g.SetPrefix("abcd")
	g.Info("测试内容")
	g.Notice("测试内容")
	g.Warning("测试内容")
	g.Error("测试内容")
	g.Critical("测试内容")
}

func t2() {
	f, err := g.Open("gylog测试", 20, 25, 2)
	if err != nil {
		fmt.Println(err)
	}
	g.SetOutput(f)
	s := "测试内容测试内容测试内容测试内容"
	for {
		<-time.After(time.Duration(1))
		g.Info(s)
	}
}
