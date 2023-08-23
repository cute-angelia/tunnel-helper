package main

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"regexp"
	"sync"
	"time"
	"tunnel-helper/internal"
)

func main() {
	app := &cli.App{
		Name: "ssh tunnel",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "name",
				Value: ".*",
				Usage: "start service by name",
			},
		},
		Action: func(cCtx *cli.Context) error {
			// 检查配置文件是否存在，不存在创建图片
			internal.CheckFileExist()
			// 检查配置
			viper := viper.New()
			viper.SetConfigName("config")
			viper.SetConfigType("json")
			viper.AddConfigPath("./")
			iconfig := internal.Config{}
			viper.ReadInConfig()
			viper.Unmarshal(&iconfig)

			// 启动服务
			startTunnels(iconfig, cCtx.String("name"))

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// errTunnel .
type errTunnel struct {
	Idx    int
	Errmsg string
}

func newErrTunnel(idx int, format string, args ...interface{}) errTunnel {
	return errTunnel{
		Idx:    idx,
		Errmsg: fmt.Sprintf(format, args...),
	}
}

func (err errTunnel) Error() string {
	return err.Errmsg
}

// support context cancel
// upport tunnel open by ident matcher
func startTunnels(cfg internal.Config, tunnelIdentPattern string) {
	var (
		errChan    = make(chan errTunnel, 1) // 异常channel
		wg         = sync.WaitGroup{}        // 同步组
		tunnelChan = make(chan int, 1)       // 运行中tunnels 计数

		exp *regexp.Regexp // pattern regexp
	)

	// compile pattern
	exp = regexp.MustCompile(tunnelIdentPattern)

	wg.Add(len(cfg.Tunnels))
	// create and ssh tunnel and goto work
	for idx, v := range cfg.Tunnels {
		if v.SSH == nil {
			v.SSH = cfg.SSH
		}

		// matche with ident pattern
		if !exp.MatchString(v.Ident) {
			log.Println(fmt.Sprintf("Warnf tunnel ident=%s, not matched with pattern=%s, so skipped", v.Ident, tunnelIdentPattern))
			continue
		}

		go func(idx int, tunnelCfg *internal.TunnelConfig, errChan chan<- errTunnel) {
			defer wg.Done()
			defer func() { tunnelChan <- -1 }()
			tunnelChan <- 1

			// // valid tunnel config
			// // has been moved to loadConfig
			// if err := tunnelCfg.Valid(); err != nil {
			// 	errChan <- newErrTunnel(idx, "invalid config, err=%v", err)
			// 	return
			// }

			// open tunnel and prepare
			tunnel := internal.NewSSHTunnel(tunnelCfg)
			if err := tunnel.Start(); err != nil {
				errChan <- newErrTunnel(idx, "tunnel broken, err=%v", err)
				return
			}
		}(idx, v, errChan)
	}

	// log errors
	go func() {
		for err := range errChan {
			log.Printf("error tunnelIdx=%d: %s", err.Idx, err.Errmsg)
		}
	}()

	// record changes of opening-tunnel count
	go func() {
		running := 0
		msg := ""
		for cntChange := range tunnelChan {
			// if runningTunnelsCnt changed to notify
			// FIXME: atomic op with running
			running += cntChange
			if cntChange >= 0 {
				// true: starting
				msg = fmt.Sprintf("%d tunnel starting, current: %d", cntChange, running)
			} else {
				// true: quit
				msg = fmt.Sprintf("%d tunnel break, current: %d", 0-cntChange, running)
			}
			log.Printf(msg)
		}
	}()

	wg.Wait()
	close(errChan)
	close(tunnelChan)
	// wait for all error message outputing
	time.Sleep(100 * time.Millisecond)
	log.Println("tunnel-helper is finished")
}
