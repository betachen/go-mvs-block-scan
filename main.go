package main

import (
	"flag"
	"github.com/inconshreveable/log15"
	"mvs_sync/bc"
	"mvs_sync/config"
	"mvs_sync/db"
	. "mvs_sync/syncmgr"
	"runtime"
	"strconv"
)

var (
	configfile     = flag.String("conf", "config.json", "Specify configuration file (default: config.json")
	beginfrom      = flag.Int64("beginfrom", 0, "Set minimum block height to sync")
	endto          = flag.Int64("endto", 0, "Set maximum block height to sync")
	goroutinecount = flag.Int("gc", 128, "Set go routine count to use for sync")
)

var log = log15.New()

func main() {
	flag.Parse()

	if !config.LoadFile(*configfile) {
		return
	}

	host := config.Get("mvsd", "host")
	port, _ := strconv.Atoi(config.Get("mvsd", "port"))
	rpcuser := config.Get("mvsd", "rpcuser")
	rpcpassword := config.Get("mvsd", "rpcpassword")
	useSSL := false
	if config.Get("mvsd", "ssl") == "true" {
		useSSL = true
	}
	log.Info("mvsd settings from configfile", "host", host, "port", port, "user", rpcuser, "password", rpcpassword, "useSSL", useSSL)

	bcclient, err := bc.New(host, port, rpcuser, rpcpassword, useSSL)
	if err != nil {
		log.Crit("create bc error: [%s]", err)
		return
	}

	dbclient := db.GetInstance()
	err = dbclient.Open()
	if err != nil {
		log.Crit("[%d] config file: %s", 2, err)
		return
	}
	defer dbclient.Close()

	syncmgr := GetInstance()
	syncmgr.Init(bcclient, dbclient, *beginfrom, *endto, *goroutinecount)

	runtime.GOMAXPROCS(runtime.NumCPU())

	done := make(chan struct{})
	syncmgr.Start()
	<-done
}
