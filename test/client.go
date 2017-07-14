package main

import (
	"bufio"
	"encoding/hex"
	"io"
	"net"
	"os"

	"go.uber.org/zap"
	cli "gopkg.in/urfave/cli.v2"

	"github.com/rssllyn/go-raknet/extended"
)

var Logger *zap.Logger

func init() {
	Logger, _ = zap.NewDevelopment()
}

func main() {
	app := &cli.App{
		Name: "raknet test server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "server-address",
				Value: "127.0.0.1:8711",
				Usage: "listening address:port",
			},
			&cli.StringFlag{
				Name:  "log-mode",
				Value: "development",
				Usage: "log mode, can be development or production",
			},
			&cli.StringFlag{
				Name:  "log-file",
				Value: "",
				Usage: "log file path, only has effect on production log mode. leave empty to disable file logging",
			},
		},
		Action: func(ctx *cli.Context) error {
			conn, err := extended.Dial(ctx.String("server-address"))
			if err != nil {
				Logger.Fatal("failed to connect to server")
			}
			Logger.Debug("connection accepted")

			go echo(conn)
			select {}
			return nil
		},
	}
	app.Run(os.Args)
}

func echo(conn net.Conn) {
	buff := make([]byte, 2000)
	scanner := bufio.NewScanner(os.Stdin)
	for true {
		var linesize int
		if scanner.Scan() {
			linesize = len(scanner.Bytes())
			if linesize == 0 {
				conn.Close()
				return
			}
			Logger.Debug("sending", zap.String("data", scanner.Text()))
			if _, err := conn.Write(scanner.Bytes()); err != nil {
				Logger.Warn("failed to send to client", zap.Error(err))
				return
			}
			if linesize > len(buff) {
				buff = make([]byte, linesize)
			}
			if _, err := io.ReadFull(conn, buff[:linesize]); err != nil {
				Logger.Warn("failed to read from client", zap.Error(err))
				return
			}
			Logger.Debug("data received from client", zap.String("hex", hex.EncodeToString(buff[:linesize])), zap.String("string", string(buff[:linesize])))
		}
	}
}
