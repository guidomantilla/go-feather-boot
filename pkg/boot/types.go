package boot

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	feather_commons_environment "github.com/guidomantilla/go-feather-commons/pkg/environment"
	feather_commons_log "github.com/guidomantilla/go-feather-commons/pkg/log"
	feather_commons_util "github.com/guidomantilla/go-feather-commons/pkg/util"
	feather_security "github.com/guidomantilla/go-feather-security/pkg/security"
	feather_sql_datasource "github.com/guidomantilla/go-feather-sql/pkg/datasource"
	feather_sql "github.com/guidomantilla/go-feather-sql/pkg/sql"
	"google.golang.org/grpc"
)

type Enablers struct {
	HttpServerEnabled bool
	GrpcServerEnabled bool
	DatabaseEnabled   bool
}

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
	AppVersion             string
	LogLevel               string
	CmdArgs                []string
	Enablers               *Enablers
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

func NewApplicationContext(appName string, version string, args []string, logger feather_commons_log.Logger, enablers *Enablers, builder *BeanBuilder) *ApplicationContext {

	if appName == "" {
		feather_commons_log.Fatal("starting up - error setting up the ApplicationContext: appName is empty")
	}

	if version == "" {
		feather_commons_log.Fatal("starting up - error setting up the ApplicationContext: version is empty")
	}

	if args == nil {
		feather_commons_log.Fatal("starting up - error setting up the ApplicationContext: args is nil")
	}

	if logger == nil {
		feather_commons_log.Fatal("starting up - error setting up the application: logger is nil")
	}

	if enablers == nil {
		feather_commons_log.Warn("starting up - warning setting up the application: http server, grpc server and database connectivity are disabled")
		enablers = &Enablers{}
	}

	if builder == nil { //nolint:staticcheck
		feather_commons_log.Fatal("starting up - error setting up the ApplicationContext: builder is nil")
	}

	ctx := &ApplicationContext{
		AppName:    appName,
		AppVersion: version,
		CmdArgs:    args,
		Logger:     logger,
		Enablers:   enablers,
		SecurityConfig: &SecurityConfig{
			TokenSignatureKey: feather_commons_util.ValueToPtr("SecretYouShouldHide"),
		},
		HttpConfig: &HttpConfig{
			Host: feather_commons_util.ValueToPtr("localhost"),
			Port: feather_commons_util.ValueToPtr("8080"),
		},
		GrpcConfig: &GrpcConfig{
			Host: feather_commons_util.ValueToPtr("localhost"),
			Port: feather_commons_util.ValueToPtr("50051"),
		},
	}

	feather_commons_log.Debug("starting up - setting up environment variables")
	ctx.Environment = builder.Environment(ctx) //nolint:staticcheck

	feather_commons_log.Debug("starting up - setting up configuration")
	builder.Config(ctx) //nolint:staticcheck

	if ctx.Enablers.DatabaseEnabled {
		feather_commons_log.Debug("starting up - setting up db connectivity")
		ctx.DatasourceContext = builder.DatasourceContext(ctx)   //nolint:staticcheck
		ctx.Datasource = builder.Datasource(ctx)                 //nolint:staticcheck
		ctx.TransactionHandler = builder.TransactionHandler(ctx) //nolint:staticcheck
	} else {
		feather_commons_log.Warn("starting up - warning setting up database configuration. database connectivity is disabled")
	}

	feather_commons_log.Debug("starting up - setting up security")
	ctx.PasswordEncoder = builder.PasswordEncoder(ctx)                                                                          //nolint:staticcheck
	ctx.PasswordGenerator = builder.PasswordGenerator(ctx)                                                                      //nolint:staticcheck
	ctx.PasswordManager = builder.PasswordManager(ctx)                                                                          //nolint:staticcheck
	ctx.PrincipalManager, ctx.TokenManager = builder.PrincipalManager(ctx), builder.TokenManager(ctx)                           //nolint:staticcheck
	ctx.AuthenticationService, ctx.AuthorizationService = builder.AuthenticationService(ctx), builder.AuthorizationService(ctx) //nolint:staticcheck
	ctx.AuthenticationEndpoint, ctx.AuthorizationFilter = builder.AuthenticationEndpoint(ctx), builder.AuthorizationFilter(ctx) //nolint:staticcheck

	if ctx.Enablers.HttpServerEnabled {
		feather_commons_log.Debug("starting up - setting up http server")
		ctx.PublicRouter, ctx.PrivateRouter = builder.HttpServer(ctx) //nolint:staticcheck
	} else {
		feather_commons_log.Warn("starting up - warning setting up http configuration. http server is disabled")
	}

	if ctx.Enablers.GrpcServerEnabled {
		feather_commons_log.Debug("starting up - setting up grpc server")
		ctx.GrpcServiceDesc, ctx.GrpcServiceServer = builder.GrpcServer(ctx) //nolint:staticcheck
	} else {
		feather_commons_log.Warn("starting up - warning setting up grpc configuration. grpc server is disabled")
	}

	return ctx
}

func (ctx *ApplicationContext) Stop() {

	var err error

	if ctx.Datasource != nil && ctx.DatasourceContext != nil {

		var database *sql.DB
		feather_commons_log.Debug("shutting down - closing up db connection")

		if database, err = ctx.Datasource.GetDatabase(); err != nil {
			feather_commons_log.Error(fmt.Sprintf("shutting down - error db connection: %s", err.Error()))
			return
		}

		if err = database.Close(); err != nil {
			feather_commons_log.Error(fmt.Sprintf("shutting down - error closing db connection: %s", err.Error()))
			return
		}

		feather_commons_log.Debug("shutting down - db connection closed")
	}

	feather_commons_log.Info(fmt.Sprintf("Application %s stopped", strings.Join([]string{ctx.AppName, ctx.AppVersion}, " - ")))
}
