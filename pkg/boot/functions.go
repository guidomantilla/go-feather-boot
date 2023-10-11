package boot

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"

	feather_commons_log "github.com/guidomantilla/go-feather-commons/pkg/log"
	feather_web_server "github.com/guidomantilla/go-feather-web/pkg/server"
	"github.com/qmdx00/lifecycle"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type InitDelegateFunc func(ctx ApplicationContext) error

func Init(appName string, version string, args []string, logger feather_commons_log.Logger, enablers *Enablers, builder *BeanBuilder, fn InitDelegateFunc) error {

	if appName == "" {
		feather_commons_log.Fatal("starting up - error setting up the application: appName is empty")
	}

	if args == nil {
		feather_commons_log.Fatal("starting up - error setting up the application: args is nil")
	}

	if logger == nil {
		feather_commons_log.Fatal("starting up - error setting up the application: logger is nil")
	}
	if enablers == nil {
		feather_commons_log.Warn("starting up - warning setting up the application: http server, grpc server and database connectivity are disabled")
		enablers = &Enablers{}
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

	ctx := NewApplicationContext(appName, version, args, logger, enablers, builder)
	defer ctx.Stop()

	if err := fn(*ctx); err != nil {
		feather_commons_log.Fatal(fmt.Sprintf("starting up - error setting up the application: %s", err.Error()))
	}

	if ctx.Enablers.HttpServerEnabled {
		if ctx.PublicRouter == nil || ctx.HttpConfig == nil || ctx.HttpConfig.Host == nil || ctx.HttpConfig.Port == nil {
			feather_commons_log.Fatal("starting up - error setting up the application: http server is enabled but no public router or http config is provided")
		}
		httpServer := &http.Server{
			Addr:              net.JoinHostPort(*ctx.HttpConfig.Host, *ctx.HttpConfig.Port),
			Handler:           ctx.PublicRouter,
			ReadHeaderTimeout: 60000,
		}
		app.Attach("HttpServer", feather_web_server.BuildHttpServer(httpServer))
	}

	if ctx.Enablers.GrpcServerEnabled {
		if ctx.GrpcServiceDesc == nil || ctx.GrpcServiceServer == nil || ctx.GrpcConfig == nil || ctx.GrpcConfig.Host == nil || ctx.GrpcConfig.Port == nil {
			feather_commons_log.Fatal("starting up - error setting up the application: grpc server is enabled but no grpc service descriptor, grpc service server or grpc config is provided")
		}
		server := grpc.NewServer()
		server.RegisterService(ctx.GrpcServiceDesc, ctx.GrpcServiceServer)
		reflection.Register(server)
		app.Attach("GrpcServer", feather_web_server.BuildGrpcServer(net.JoinHostPort(*ctx.GrpcConfig.Host, *ctx.GrpcConfig.Port), server))
	}

	feather_commons_log.Info(fmt.Sprintf("Application %s started", strings.Join([]string{appName, version}, " - ")))
	return app.Run()
}
