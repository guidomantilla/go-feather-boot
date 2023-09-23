package boot

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	feather_commons_environment "github.com/guidomantilla/go-feather-commons/pkg/environment"
	feather_commons_log "github.com/guidomantilla/go-feather-commons/pkg/log"
	feather_security "github.com/guidomantilla/go-feather-security/pkg/security"
	feather_sql_datasource "github.com/guidomantilla/go-feather-sql/pkg/datasource"
	feather_sql "github.com/guidomantilla/go-feather-sql/pkg/sql"
	"google.golang.org/grpc"
)

type HttpConfig struct {
	Host            *string
	Port            *string
	SwaggerPort     *string
	CorsAllowOrigin *string
}

type GrpcConfig struct {
	Host *string
	Port *string
}

type SecurityConfig struct {
	TokenSignatureKey       *string
	PasswordMinSpecialChars *string
	PasswordMinNumber       *string
	PasswordMinUpperCase    *string
	PasswordLength          *string
}

type DatabaseConfig struct {
	ParamHolder        feather_sql.ParamHolder
	Driver             feather_sql.DriverName
	DatasourceUrl      *string
	DatasourceUsername *string
	DatasourcePassword *string
	DatasourceServer   *string
	DatasourceService  *string
}

type ApplicationContext struct {
	AppName                string
	LogLevel               string
	CmdArgs                []string
	HttpConfig             *HttpConfig
	GrpcConfig             *GrpcConfig
	SecurityConfig         *SecurityConfig
	DatabaseConfig         *DatabaseConfig
	Logger                 feather_commons_log.Logger
	Environment            feather_commons_environment.Environment
	DatasourceContext      feather_sql_datasource.DatasourceContext
	Datasource             feather_sql_datasource.Datasource
	TransactionHandler     feather_sql_datasource.TransactionHandler
	PasswordEncoder        feather_security.PasswordEncoder
	PasswordGenerator      feather_security.PasswordGenerator
	PasswordManager        feather_security.PasswordManager
	PrincipalManager       feather_security.PrincipalManager
	TokenManager           feather_security.TokenManager
	AuthenticationService  feather_security.AuthenticationService
	AuthenticationEndpoint feather_security.AuthenticationEndpoint
	AuthorizationService   feather_security.AuthorizationService
	AuthorizationFilter    feather_security.AuthorizationFilter
	PublicRouter           *gin.Engine
	PrivateRouter          *gin.RouterGroup
	GrpcServiceDesc        *grpc.ServiceDesc
	GrpcServiceServer      any
}

func NewApplicationContext(appName string, args []string, logger feather_commons_log.Logger, builder *BeanBuilder) *ApplicationContext {

	if appName == "" {
		feather_commons_log.Fatal("starting up - error setting up the ApplicationContext: appName is empty")
	}

	feather_commons_log.Info(fmt.Sprintf("starting up - starting up ApplicationContext %s", appName))

	if args == nil {
		feather_commons_log.Fatal("starting up - error setting up the ApplicationContext: args is nil")
	}

	if logger == nil {
		feather_commons_log.Fatal("starting up - error setting up the application: logger is nil")
	}

	if builder == nil { //nolint:staticcheck
		feather_commons_log.Fatal("starting up - error setting up the ApplicationContext: builder is nil")
	}

	ctx := &ApplicationContext{}
	ctx.AppName, ctx.CmdArgs, ctx.Logger = appName, args, logger

	feather_commons_log.Info("starting up - setting up environment variables")
	ctx.Environment = builder.Environment(ctx) //nolint:staticcheck

	feather_commons_log.Info("starting up - setting up configuration")
	builder.Config(ctx) //nolint:staticcheck

	feather_commons_log.Info("starting up - setting up DB connection")
	ctx.DatasourceContext = builder.DatasourceContext(ctx)   //nolint:staticcheck
	ctx.Datasource = builder.Datasource(ctx)                 //nolint:staticcheck
	ctx.TransactionHandler = builder.TransactionHandler(ctx) //nolint:staticcheck

	feather_commons_log.Info("starting up - setting up security")
	ctx.PasswordEncoder = builder.PasswordEncoder(ctx)                                                                          //nolint:staticcheck
	ctx.PasswordGenerator = builder.PasswordGenerator(ctx)                                                                      //nolint:staticcheck
	ctx.PasswordManager = builder.PasswordManager(ctx)                                                                          //nolint:staticcheck
	ctx.PrincipalManager, ctx.TokenManager = builder.PrincipalManager(ctx), builder.TokenManager(ctx)                           //nolint:staticcheck
	ctx.AuthenticationService, ctx.AuthorizationService = builder.AuthenticationService(ctx), builder.AuthorizationService(ctx) //nolint:staticcheck
	ctx.AuthenticationEndpoint, ctx.AuthorizationFilter = builder.AuthenticationEndpoint(ctx), builder.AuthorizationFilter(ctx) //nolint:staticcheck

	feather_commons_log.Info("starting up - setting up http server")
	ctx.PublicRouter, ctx.PrivateRouter = builder.HttpServer(ctx) //nolint:staticcheck

	feather_commons_log.Info("starting up - setting up grpc server")
	ctx.GrpcServiceDesc, ctx.GrpcServiceServer = builder.GrpcServer(ctx) //nolint:staticcheck

	return ctx
}

func (ctx *ApplicationContext) Stop() {

	var err error

	if ctx.Datasource != nil && ctx.DatasourceContext != nil {

		var database *sql.DB
		feather_commons_log.Info("shutting down - closing up db connection")

		if database, err = ctx.Datasource.GetDatabase(); err != nil {
			feather_commons_log.Error(fmt.Sprintf("shutting down - error db connection: %s", err.Error()))
			return
		}

		if err = database.Close(); err != nil {
			feather_commons_log.Error(fmt.Sprintf("shutting down - error closing db connection: %s", err.Error()))
			return
		}

		feather_commons_log.Info("shutting down - db connection closed")
	}

	feather_commons_log.Info(fmt.Sprintf("shutting down - ApplicationContext closed %s", ctx.AppName))
}
