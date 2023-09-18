package boot

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	feather_commons_environment "github.com/guidomantilla/go-feather-commons/pkg/environment"
	feather_security "github.com/guidomantilla/go-feather-security/pkg/security"
	feather_sql_datasource "github.com/guidomantilla/go-feather-sql/pkg/datasource"
	feather_sql "github.com/guidomantilla/go-feather-sql/pkg/sql"
	feather_sql_transaction "github.com/guidomantilla/go-feather-sql/pkg/transaction"
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
	TokenSignatureKey *string
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
	CmdArgs                []string
	HttpConfig             *HttpConfig
	GrpcConfig             *GrpcConfig
	SecurityConfig         *SecurityConfig
	DatabaseConfig         *DatabaseConfig
	Environment            feather_commons_environment.Environment
	DatasourceContext      feather_sql_datasource.DatasourceContext
	Datasource             feather_sql_datasource.Datasource
	TransactionHandler     feather_sql_transaction.TransactionHandler
	PasswordEncoder        feather_security.PasswordEncoder
	PasswordGenerator      feather_security.PasswordGenerator
	PasswordManager        feather_security.PasswordManager
	PrincipalManager       feather_security.PrincipalManager
	TokenManager           feather_security.TokenManager
	AuthenticationService  feather_security.AuthenticationService
	AuthenticationEndpoint feather_security.AuthenticationEndpoint
	AuthorizationService   feather_security.AuthorizationService
	AuthorizationFilter    feather_security.AuthorizationFilter
	Router                 *gin.Engine
	SecureRouter           *gin.RouterGroup
}

func NewApplicationContext(appName string, args []string, builder *BeanBuilder) *ApplicationContext {

	if appName == "" {
		slog.Error("starting up - error setting up the ApplicationContext: appName is empty")
		os.Exit(1)
	}

	slog.Info(fmt.Sprintf("starting up - starting up ApplicationContext %s", appName))

	if args == nil {
		slog.Error("starting up - error setting up the ApplicationContext: args is nil")
		os.Exit(1)
	}

	if builder == nil {
		slog.Error("starting up - error setting up the ApplicationContext: builder is nil")
		os.Exit(1)
	}

	ctx := &ApplicationContext{}
	ctx.AppName, ctx.CmdArgs = appName, args

	slog.Info("starting up - setting up environment variables")
	ctx.Environment = builder.Environment(ctx)

	slog.Info("starting up - setting up configuration")
	builder.Config(ctx)

	slog.Info("starting up - setting up DB connection")
	ctx.DatasourceContext = builder.DatasourceContext(ctx)
	ctx.Datasource = builder.Datasource(ctx)
	ctx.TransactionHandler = builder.TransactionHandler(ctx)

	slog.Info("starting up - setting up security")
	ctx.PasswordEncoder = builder.PasswordEncoder(ctx)
	ctx.PasswordGenerator = builder.PasswordGenerator(ctx)
	ctx.PasswordManager = builder.PasswordManager(ctx)
	ctx.PrincipalManager, ctx.TokenManager = builder.PrincipalManager(ctx), builder.TokenManager(ctx)
	ctx.AuthenticationService, ctx.AuthorizationService = builder.AuthenticationService(ctx), builder.AuthorizationService(ctx)
	ctx.AuthenticationEndpoint, ctx.AuthorizationFilter = builder.AuthenticationEndpoint(ctx), builder.AuthorizationFilter(ctx)

	ctx.Router = gin.Default()

	return ctx
}

func (ctx *ApplicationContext) Stop() {

	var err error

	if ctx.Datasource != nil && ctx.DatasourceContext != nil {

		var database *sql.DB
		slog.Info("shutting down - closing up db connection")

		if database, err = ctx.Datasource.GetDatabase(); err != nil {
			slog.Error(fmt.Sprintf("shutting down - error db connection: %s", err.Error()))
			return
		}

		if err = database.Close(); err != nil {
			slog.Error(fmt.Sprintf("shutting down - error closing db connection: %s", err.Error()))
			return
		}

		slog.Info("shutting down - db connection closed")
	}

	slog.Info(fmt.Sprintf("shutting down - ApplicationContext closed %s", ctx.AppName))
}
