package boot

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"

	feather_web_server "github.com/guidomantilla/go-feather-web/pkg/server"
	"github.com/qmdx00/lifecycle"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type InitDelegateFunc func(ctx ApplicationContext) error

func Init(appName string, version string, args []string, builder *BeanBuilder, fn InitDelegateFunc) error {

	if appName == "" {
		slog.Error("starting up - error setting up the application: appName is empty")
		os.Exit(1)
	}

	if args == nil {
		slog.Error("starting up - error setting up the application: args is nil")
		os.Exit(1)
	}

	if builder == nil {
		slog.Error("starting up - error setting up the application: builder is nil")
		os.Exit(1)
	}

	if fn == nil {
		slog.Error("starting up - error setting up the application: fn is nil")
		os.Exit(1)
	}

	app := lifecycle.NewApp(
		lifecycle.WithName(appName),
		lifecycle.WithVersion(version),
		lifecycle.WithSignal(syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL),
	)

	ctx := NewApplicationContext(strings.Join([]string{appName, version}, " - "), args, builder)
	defer ctx.Stop()

	if err := fn(*ctx); err != nil {
		slog.Error("starting up - error setting up the application.", "message", err.Error())
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:              net.JoinHostPort(*ctx.HttpConfig.Host, *ctx.HttpConfig.Port),
		Handler:           ctx.PublicRouter,
		ReadHeaderTimeout: 60000,
	}

	app.Attach("HttpServer", feather_web_server.BuildHttpServer(httpServer))

	if ctx.GrpcConfig != nil {
		server := grpc.NewServer()
		server.RegisterService(ctx.GrpcServiceDesc, ctx.GrpcServiceServer)
		reflection.Register(server)
		app.Attach("GrpcServer", feather_web_server.BuildGrpcServer(net.JoinHostPort(*ctx.GrpcConfig.Host, *ctx.GrpcConfig.Port), server))
	}

	return app.Run()
}
