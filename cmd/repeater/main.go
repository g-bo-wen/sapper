package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zssky/log"

	"github.com/dearcode/sapper/repeater"
	"github.com/dearcode/sapper/repeater/config"
	"github.com/dearcode/sapper/util"
)

var (
	addr    = flag.String("h", ":9000", "api listen address")
	debug   = flag.Bool("debug", false, "debug write log to console.")
	version = flag.Bool("v", false, "show version info")
)

func main() {
	flag.Parse()

	if *version {
		util.PrintVersion()
		return
	}

	if !*debug {
		log.SetOutputByName("./logs/api.log")
		log.SetHighlighting(false)
		log.SetRotateByDay()
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		panic(err.Error())
	}

	if err = repeater.ServerInit(); err != nil {
		panic(err.Error())
	}

	as := http.Server{Handler: repeater.Server}

	go func() {
		if err = as.Serve(ln); err != nil {
			log.Error(err)
		}
	}()

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGUSR1)

	s := <-shutdown
	log.Warningf("recv signal %v, close.", s)
	as.Shutdown(context.Background())
	time.Sleep(time.Duration(config.Repeater.Cache.Timeout) * time.Second)
	log.Warningf("server exit")
}
