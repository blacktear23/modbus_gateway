package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/blacktear23/modbus_gateway/config"
	"github.com/blacktear23/modbus_gateway/server"
)

var (
	VERSION    = "1.0.0"
	BUILD_TIME = ""
)

func printVersion() {
	fmt.Printf("Version: %s\n", VERSION)
	if BUILD_TIME != "" {
		fmt.Printf("Build At: %s\n", BUILD_TIME)
	}
}

func main() {
	var (
		listenAddr string
		configFile string
		timeout    int
		version    bool
	)

	flag.StringVar(&listenAddr, "l", ":502", "Modbus TCP server listen address")
	flag.StringVar(&configFile, "c", "config.yaml", "Config file name")
	flag.IntVar(&timeout, "t", 0, "Timeout unit is ms")
	flag.BoolVar(&version, "v", false, "Show version")
	flag.Parse()

	if version {
		printVersion()
		return
	}

	_, err := net.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		fmt.Println("Invalid listen address")
		return
	}
	cfg, err := config.NewConfig(configFile)
	if err != nil {
		fmt.Println("Load config file got error:", err)
		return
	}

	router := server.NewRouter(cfg)
	server := server.NewTCPServer(listenAddr, timeout, router)
	err = server.Start()
	if err != nil {
		fmt.Println("Cannot start TCP server:", err)
		return
	}
	fmt.Println("Start Modbus TCP server at", listenAddr)
	WaitSignal(func() {
		err := cfg.Reload()
		if err != nil {
			log.Println("Reload config file error:", err)
			return
		}
		router.Reload()
	}, nil)
}

type SignalCallback func()

func WaitSignal(onReload, onExit SignalCallback) {
	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGHUP)
	for sig := range sigChan {
		if sig == syscall.SIGHUP {
			// Reload resolve rule file
			if onReload != nil {
				onReload()
			}
		} else {
			if onExit != nil {
				onExit()
			}
			log.Fatal("Server Exit")
		}
	}
}
