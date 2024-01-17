package main

import (
	"encoding/json"
	"fmt"
	"github.com/civet148/log"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"os/signal"
	"time"
)

const (
	Version     = "v0.1.0"
	ProgramName = "portmap"
)

var (
	BuildTime = "2024-01-16"
	GitCommit = ""
)

const (
	CMD_NAME_START = "start"
	CMD_NAME_RUN   = "run"
)

const (
	CMD_FLAG_NAME_DEBUG  = "debug"
	CMD_FLAG_NAME_CONFIG = "config"
)

const (
	DEFAULT_CONFIG_FILE = "config.json"
)

func init() {
	log.SetLevel("info")
}

func grace() {
	//capture signal of Ctrl+C and gracefully exit
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt)
	go func() {
		for {
			select {
			case s := <-sigChannel:
				{
					if s != nil && s == os.Interrupt {
						fmt.Printf("Ctrl+C signal captured, program exiting...\n")
						close(sigChannel)
						os.Exit(0)
					}
				}
			}
		}
	}()
}

func main() {
	grace()
	commands := []*cli.Command{
		cmdStart,
	}

	app := &cli.App{
		Name:     ProgramName,
		Version:  fmt.Sprintf("%s %s commit %s", Version, BuildTime, GitCommit),
		Flags:    []cli.Flag{},
		Commands: commands,
		Action:   nil,
	}
	if err := app.Run(os.Args); err != nil {
		log.Errorf("exit in error %s", err)
		os.Exit(1)
		return
	}
}

var cmdStart = &cli.Command{
	Name: CMD_NAME_START,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    CMD_FLAG_NAME_DEBUG,
			Aliases: []string{"d"},
			Usage:   "open debug mode",
		},
		&cli.StringFlag{
			Name:    CMD_FLAG_NAME_CONFIG,
			Aliases: []string{"c"},
			Usage:   "config file path",
		},
	},
	Action: func(cctx *cli.Context) error {
		return Start(cctx)
	},
}

func Start(cctx *cli.Context) error {
	var elems []ConfigElement
	strConfig := DEFAULT_CONFIG_FILE
	if cctx.IsSet(CMD_FLAG_NAME_CONFIG) {
		strConfig = cctx.String(CMD_FLAG_NAME_CONFIG)
	}
	if cctx.IsSet(CMD_FLAG_NAME_DEBUG) {
		log.SetLevel("debug")
	} else {
		log.SetLevel("fatal")
	}
	bs, err := ioutil.ReadFile(strConfig)
	if err != nil {
		return log.Errorf(err.Error())
	}
	_ = bs
	err = json.Unmarshal(bs, &elems)
	if err != nil {
		return log.Errorf(err.Error())
	}
	CreateForwards(elems)
	time.Sleep(3 * time.Second)
	PrintForwards(elems)
	var c = make(chan bool, 1)
	<-c //block main go routine
	return nil
}

func CreateForwards(elems []ConfigElement) {
	var bridges = make(map[string]*NetBridge)
	for _, e := range elems {
		if !e.Enable {
			continue
		}
		if _, ok := bridges[e.Name]; ok {
			log.Panic("config element name %s already exists", e.Name)
		}
		bridge := NewNetBridge(e)
		bridges[e.Name] = bridge
	}
}

func PrintForwards(elems []ConfigElement) {
	log.Printf("-------------------------------------------------------")
	for _, e := range elems {
		if !e.Enable {
			continue
		}
		log.Printf("|  %-5v => %-30v %-10s |", e.Local, e.Remote, e.Name)
	}
	log.Printf("-------------------------------------------------------")
}
