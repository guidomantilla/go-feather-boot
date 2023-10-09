package boot

import (
	"net"
	"net/http"
	"syscall"

	feather_commons_log "github.com/guidomantilla/go-feather-commons/pkg/log"
	feather_web_server "github.com/guidomantilla/go-feather-web/pkg/server"
	"github.com/qmdx00/lifecycle"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type InitDelegateFunc func(ctx ApplicationContext) error

func Init(appName string, version string, args []string, logger feather_commons_log.Logger, builder *BeanBuilder, fn InitDelegateFunc) error {

	if appName == "" {
		feather_commons_log.Fatal("starting up - error setting up the application: appName is empty")
	}

	if args == nil {
		feather_commons_log.Fatal("starting up - error setting up the application: args is nil")
	}

	if logger == nil {
		feather_commons_log.Fatal("starting up - error setting up the application: logger is nil")
	}

	if builder == nil {
		feather_commons_log.Fatal("starting up - error setting up the application: builder is nil")
	}

	if fn == nil {
		feather_commons_log.Fatal("starting up - error setting up the application: fn is nil")
	}

	app := lifecycle.NewApp(
		lifecycle.WithName(appName),
		lifecycle.WithVersion(version),
		lifecycle.WithSignal(syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL),
	)

	ctx := NewApplicationContext(appName, version, args, logger, builder)
	defer ctx.Stop()

	if err := fn(*ctx); err != nil {
		feather_commons_log.Fatal("starting up - error setting up the application.", "message", err.Error())
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
