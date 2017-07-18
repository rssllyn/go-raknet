package main

import (
	"encoding/hex"
	"io"
	"net"
	"os"

	"go.uber.org/zap"
	cli "gopkg.in/urfave/cli.v2"

	"github.com/rssllyn/go-raknet"
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
				Name:  "listen",
				Value: ":8711",
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
			l, err := raknet.Listen(ctx.String("listen"), 1000)
			if err != nil {
				Logger.Warn("failed to listen server address")
			}
			for true {
				conn, err := l.Accept()
				if err != nil {
					Logger.Warn("failed to accept client connection", zap.Error(err))
					continue
				}
				go handleClient(conn)
			}
			return nil
		},
	}
	app.Run(os.Args)
}

func handleClient(conn net.Conn) {
	buff := make([]byte, 100)
	for true {
		n, err := conn.Read(buff)
		if err == io.EOF {
			Logger.Debug("connection closed by remote peer")
			return
		} else if err != nil {
			Logger.Warn("failed to read from client", zap.Error(err))
			return
		}
		Logger.Debug("data received from client", zap.String("hex", hex.EncodeToString(buff[:n])), zap.String("string", string(buff[:n])))
		if _, err := conn.Write(buff[:n]); err != nil {
			Logger.Warn("failed to send to client", zap.Error(err))
			return
		}
	}
}
