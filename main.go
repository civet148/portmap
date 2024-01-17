package main

import (
	"encoding/json"
	"fmt"
	"github.com/civet148/log"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"os/signal"
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
	CMD_FLAG_NAME_DEBUG   = "debug"
	CMD_FLAG_NAME_CONFIG  = "config"
	CMD_FLAG_NAME_VERBOSE = "verbose"
	CMD_FLAG_NAME_NAME    = "name"
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
	app := &cli.App{
		Name:    ProgramName,
		Usage:   "a port forwarding CLI tool",
		Version: fmt.Sprintf("%s %s commit %s", Version, BuildTime, GitCommit),
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
				Value:   "config.json",
			},
			&cli.BoolFlag{
				Name:    CMD_FLAG_NAME_VERBOSE,
				Aliases: []string{"V"},
				Usage:   "print verbose",
				Value:   false,
			},
			&cli.StringFlag{
				Name:    CMD_FLAG_NAME_NAME,
				Aliases: []string{"n"},
				Usage:   "bridge name",
			},
		},
		Action: func(cctx *cli.Context) error {
			return Start(cctx)
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Errorf("exit in error [%s]", err)
		fmt.Println(`
[
  {
    "enable": true,
    "name": "mysql",
    "local": 33306,
    "remote": "tcp://172.27.205.246:3306"
  },
  {
    "enable": true,
    "name": "postgres",
    "local": 65432,
    "remote": "tcp://172.27.205.246:5432"
  },
  {
    "enable": true,
    "name": "ssh",
    "local": 2222,
    "remote": "tcp://172.27.205.246:22"
  }
]
`)
		os.Exit(1)
		return
	}
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
	elems = LoadConfig(cctx, strConfig)
	CreateForwards(cctx, elems)
	PrintForwards(cctx, elems)
	var c = make(chan bool, 1)
	<-c //block main go routine
	return nil
}

func LoadConfig(cctx *cli.Context, strConfig string) (elems []ConfigElement) {
	bs, err := ioutil.ReadFile(strConfig)
	if err != nil {
		log.Panic(err.Error())
	}
	_ = bs
	err = json.Unmarshal(bs, &elems)
	if err != nil {
		log.Panic(err.Error())
	}
	return
}

func CreateForwards(cctx *cli.Context, elems []ConfigElement) {
	var bridges = make(map[string]*NetBridge)
	for _, e := range elems {
		if !e.Enable {
			continue
		}
		if _, ok := bridges[e.Name]; ok {
			log.Panic("config element name %s already exists", e.Name)
		}
		bridge := NewNetBridge(cctx, e)
		bridges[e.Name] = bridge
	}
}

func PrintForwards(cctx *cli.Context, elems []ConfigElement) {
	log.Printf("-------------------------------------------------------------------")
	for _, e := range elems {
		if !e.Enable {
			continue
		}
		log.Printf("|  %-5v => %-30v %-24s |", e.Local, e.Remote, e.Name)
	}
	log.Printf("-------------------------------------------------------------------")
}
