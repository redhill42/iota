package cmds

import (
	"net"
	"os"
	"os/signal"
	prof "runtime"
	"sync/atomic"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/redhill42/iota/api/mqtt"
	"github.com/redhill42/iota/api/server"
	"github.com/redhill42/iota/api/server/middleware"
	"github.com/redhill42/iota/api/server/router/devices"
	"github.com/redhill42/iota/api/server/router/system"
	"github.com/redhill42/iota/auth"
	"github.com/redhill42/iota/auth/userdb"
	"github.com/redhill42/iota/device"
)

const _CONTEXT_ROOT = "/api"

func (cli *ServerCli) CmdAPIServer(args ...string) (err error) {
	var addr string

	cmd := cli.Subcmd("apiserver")
	cmd.StringVar(&addr, []string{"-bind"}, ":8080", "API server bind address")
	cmd.ParseFlags(args, true)

	stopc := make(chan bool)
	defer close(stopc)

	users, err := userdb.Open()
	if err != nil {
		return err
	}
	defer users.Close()

	authz, err := auth.NewAuthenticator(users)
	if err != nil {
		return err
	}

	devmgr, err := device.NewDeviceManager()
	if err != nil {
		return err
	}
	defer devmgr.Close()

	api := server.New(_CONTEXT_ROOT)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	api.Accept(addr, l)

	// Initialize middlewares
	api.UseMiddleware(middleware.NewVersionMiddleware())
	api.UseMiddleware(middleware.NewAuthMiddleware(authz, devmgr, _CONTEXT_ROOT))

	// Initialize routers
	api.InitRouter(
		system.NewRouter(authz),
		devices.NewRouter(devmgr),
	)

	// Route MQTT request to API server
	broker, err := mqtt.NewBroker(api.Mux)
	if err != nil {
		return err
	}
	defer broker.Close()

	// The serve API routine never exists unless an error occurs
	// we need to start it as a goroutine and wait on it so
	// daemon doesn't exit
	waitChan := make(chan error)
	go api.Wait(waitChan)
	trapSignals(func() {
		api.Close()
		<-stopc // wait for CmdServer to return
	})

	// Server is fully initialized and handling API traffic.
	// Wait for serve API to complete.
	apiErr := <-waitChan
	if apiErr != nil {
		logrus.WithError(apiErr).Error("API server error")
	}
	logrus.Info("API server terminated")
	return nil
}

func trapSignals(cleanup func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		var interruptCount uint32
		for sig := range c {
			go func(sig os.Signal) {
				logrus.Infof("*%s*", sig)
				switch sig {
				case os.Interrupt, syscall.SIGTERM:
					if atomic.LoadUint32(&interruptCount) < 3 {
						// Initiate the cleanup only once
						if atomic.AddUint32(&interruptCount, 1) == 1 {
							// Call the provided cleanup handler
							cleanup()
							os.Exit(0)
						} else {
							return
						}
					} else {
						// 3 SIGTERM/INT signals received; force exit without cleanup
						logrus.Warnf("Forcing shutdown without cleanup")
						os.Exit(128)
					}
				case syscall.SIGQUIT:
					dumpStacks()
				}
			}(sig)
		}
	}()
}

func dumpStacks() {
	var buf []byte
	var stackSize int
	bufferLen := 16384
	for stackSize == len(buf) {
		buf = make([]byte, bufferLen)
		stackSize = prof.Stack(buf, true)
		bufferLen *= 2
	}
	buf = buf[:stackSize]
	logrus.Infof("=== BEGIN goroutine stack dump ===\n%s\n=== END goroutine stack dump ===", buf)
}
