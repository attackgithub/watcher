package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/zssky/log"

	"github.com/dearcode/watcher/alertor"
	_ "github.com/dearcode/watcher/alertor/mail"
	_ "github.com/dearcode/watcher/alertor/message"
	"github.com/dearcode/watcher/config"
	"github.com/dearcode/watcher/editor"
	_ "github.com/dearcode/watcher/editor/json"
	_ "github.com/dearcode/watcher/editor/regexp"
	_ "github.com/dearcode/watcher/editor/remove"
	_ "github.com/dearcode/watcher/editor/sqlhandle"
	"github.com/dearcode/watcher/harvester"
	_ "github.com/dearcode/watcher/harvester/kafka"
	"github.com/dearcode/watcher/meta"
	"github.com/dearcode/watcher/processor"
)

func main() {
	if err := config.Init(); err != nil {
		panic(err.Error())
	}

	if err := editor.Init(); err != nil {
		panic(err.Error())
	}

	if err := harvester.Init(); err != nil {
		panic(err.Error())
	}

	if err := processor.Init(); err != nil {
		panic(err.Error())
	}

	if err := alertor.Init(); err != nil {
		panic(err.Error())
	}

	reader := harvester.Reader()
	//正式应该多线程
	for i := 0; i < 10; i++ {
		go worker(reader)
	}

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGUSR1, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)

	s := <-shutdown
	log.Warningf("recv signal %v, close.", s)
	harvester.Stop()
	log.Warningf("server exit")
}

func worker(reader <-chan *meta.Message) {
	/*
		m := meta.NewMessage("sql", `I0426 11:21:40.488165      39 sql_log.go:54] json_data:{"name":"mysql_rw","addr":"192.168.81.31:48790","sql":"update fms_freight set id=1 where id=2","sendQueryDate":"17-4-26 11:21:40.482482612","recvResultDate":"17-4-26 11:21:40.488147166","sqlExecDuration":5664554,"datanodes":[{"name":"-50","tabletType":1,"idx":1,"sendDate":"17-4-26 11:21:40.482482612","recvDate":"17-4-26 11:21:40.488141224","shardExecuteTime":5658612}]}`)
	*/
	for msg := range reader {
		run(msg)
		//	log.Infof("msg trace:%v", msg.TraceStack())
	}
	/*
		run(m)
		log.Debugf("trace:%v", m.TraceStack())
	*/
}

func run(msg *meta.Message) {
	msg.Trace(meta.StageEditor, "begin", msg.Source)
	if err := editor.Run(msg); err != nil {
		log.Errorf("editor run error:%v", err)
		log.Error(msg.TraceStack())
		return
	}

	msg.Trace(meta.StageProcessor, "begin", "")
	ac, err := processor.Run(msg)
	if err != nil {
		if err == processor.ErrNoMatch {
			return
		}
		log.Errorf("processor run error:%v", err)
		log.Error(msg.TraceStack())
		return
	}

	msg.Trace(meta.StageAlertor, "begin", "")
	if err = alertor.Run(msg, ac); err != nil {
		log.Errorf("alertor run error:%v", err)
		log.Error(msg.TraceStack())
		return
	}
	//	msg.Trace(meta.StageAlertor, "end", "OK")

}
