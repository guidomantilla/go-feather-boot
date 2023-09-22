package boot

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	feather_commons_environment "github.com/guidomantilla/go-feather-commons/pkg/environment"
	feather_security "github.com/guidomantilla/go-feather-security/pkg/security"
	feather_sql_datasource "github.com/guidomantilla/go-feather-sql/pkg/datasource"
	feather_web_rest "github.com/guidomantilla/go-feather-web/pkg/rest"
	"google.golang.org/grpc"
)

type EnvironmentBuilderFunc func(appCtx *ApplicationContext) feather_commons_environment.Environment

type ConfigLoaderFunc func(appCtx *ApplicationContext)

type DatasourceContextBuilderFunc func(appCtx *ApplicationContext) feather_sql_datasource.DatasourceContext

type DatasourceBuilderFunc func(appCtx *ApplicationContext) feather_sql_datasource.Datasource

type TransactionHandlerBuilderFunc func(appCtx *ApplicationContext) feather_sql_datasource.TransactionHandler

type PasswordGeneratorBuilderFunc func(appCtx *ApplicationContext) feather_security.PasswordGenerator

type PasswordEncoderBuilderFunc func(appCtx *ApplicationContext) feather_security.PasswordEncoder

type PasswordManagerBuilderFunc func(appCtx *ApplicationContext) feather_security.PasswordManager

type PrincipalManagerBuilderFunc func(appCtx *ApplicationContext) feather_security.PrincipalManager

type TokenManagerBuilderFunc func(appCtx *ApplicationContext) feather_security.TokenManager

type AuthenticationServiceBuilderFunc func(appCtx *ApplicationContext) feather_security.AuthenticationService

type AuthorizationServiceBuilderFunc func(appCtx *ApplicationContext) feather_security.AuthorizationService

type AuthenticationEndpointBuilderFunc func(appCtx *ApplicationContext) feather_security.AuthenticationEndpoint

type AuthorizationFilterBuilderFunc func(appCtx *ApplicationContext) feather_security.AuthorizationFilter

type HttpServerBuilderFunc func(appCtx *ApplicationContext) (*gin.Engine, *gin.RouterGroup)

type GrpcServerBuilderFunc func(appCtx *ApplicationContext) (*grpc.ServiceDesc, any)

type BeanBuilder struct {
	Config                 ConfigLoaderFunc
	Environment            EnvironmentBuilderFunc
	DatasourceContext      DatasourceContextBuilderFunc
	Datasource             DatasourceBuilderFunc
	TransactionHandler     TransactionHandlerBuilderFunc
	PasswordEncoder        PasswordEncoderBuilderFunc
	PasswordGenerator      PasswordGeneratorBuilderFunc
	PasswordManager        PasswordManagerBuilderFunc
	PrincipalManager       PrincipalManagerBuilderFunc
	TokenManager           TokenManagerBuilderFunc
	AuthenticationService  AuthenticationServiceBuilderFunc
	AuthorizationService   AuthorizationServiceBuilderFunc
	AuthenticationEndpoint AuthenticationEndpointBuilderFunc
	AuthorizationFilter    AuthorizationFilterBuilderFunc
	HttpServer             HttpServerBuilderFunc
	GrpcServer             GrpcServerBuilderFunc
}

func NewBeanBuilder(ctx context.Context) *BeanBuilder {

	return &BeanBuilder{

		Environment: func(appCtx *ApplicationContext) feather_commons_environment.Environment {
			osArgs := os.Environ()
			return feather_commons_environment.NewDefaultEnvironment(feather_commons_environment.WithArrays(&osArgs, &appCtx.CmdArgs))
		},
		Config: func(appCtx *ApplicationContext) {
			slog.Error("starting up - error setting up configuration.", "message", "config function not implemented")
			os.Exit(1)
		},
		DatasourceContext: func(appCtx *ApplicationContext) feather_sql_datasource.DatasourceContext {
			return feather_sql_datasource.NewDefaultDatasourceContext(appCtx.DatabaseConfig.Driver, appCtx.DatabaseConfig.ParamHolder, *appCtx.DatabaseConfig.DatasourceUrl,
				*appCtx.DatabaseConfig.DatasourceUsername, *appCtx.DatabaseConfig.DatasourcePassword, *appCtx.DatabaseConfig.DatasourceServer, *appCtx.DatabaseConfig.DatasourceService)
		},
		Datasource: func(appCtx *ApplicationContext) feather_sql_datasource.Datasource {
			return feather_sql_datasource.NewDefaultDatasource(appCtx.DatasourceContext, sql.Open)
		},
		TransactionHandler: func(appCtx *ApplicationContext) feather_sql_datasource.TransactionHandler {
			return feather_sql_datasource.NewTransactionHandler(appCtx.Datasource)
		},
		PasswordEncoder: func(appCtx *ApplicationContext) feather_security.PasswordEncoder {
			return feather_security.NewBcryptPasswordEncoder()
		},
		PasswordGenerator: func(appCtx *ApplicationContext) feather_security.PasswordGenerator {
			return feather_security.NewDefaultPasswordGenerator()
		},
		PasswordManager: func(appCtx *ApplicationContext) feather_security.PasswordManager {
			return feather_security.NewDefaultPasswordManager(appCtx.PasswordEncoder, appCtx.PasswordGenerator)
		},
		PrincipalManager: func(appCtx *ApplicationContext) feather_security.PrincipalManager {
			return feather_security.NewInMemoryPrincipalManager(appCtx.PasswordManager)
		},
		TokenManager: func(appCtx *ApplicationContext) feather_security.TokenManager {
			return feather_security.NewJwtTokenManager([]byte(*appCtx.SecurityConfig.TokenSignatureKey), feather_security.WithIssuer(appCtx.AppName))
		},
		AuthenticationService: func(appCtx *ApplicationContext) feather_security.AuthenticationService {
			return feather_security.NewDefaultAuthenticationService(appCtx.PasswordManager, appCtx.PrincipalManager, appCtx.TokenManager)
		},
		AuthorizationService: func(appCtx *ApplicationContext) feather_security.AuthorizationService {
			return feather_security.NewDefaultAuthorizationService(appCtx.TokenManager, appCtx.PrincipalManager)
		},
		AuthenticationEndpoint: func(appCtx *ApplicationContext) feather_security.AuthenticationEndpoint {
			return feather_security.NewDefaultAuthenticationEndpoint(appCtx.AuthenticationService)
		},
		AuthorizationFilter: func(appCtx *ApplicationContext) feather_security.AuthorizationFilter {
			return feather_security.NewDefaultAuthorizationFilter(appCtx.AuthorizationService)
		},
		HttpServer: func(appCtx *ApplicationContext) (*gin.Engine, *gin.RouterGroup) {
			engine := gin.Default()
			engine.POST("/login", appCtx.AuthenticationEndpoint.Authenticate)
			engine.GET("/health", func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{"status": "alive"})
			})
			engine.NoRoute(func(c *gin.Context) {
				c.JSON(http.StatusNotFound, feather_web_rest.NotFoundException("resource not found"))
			})
			engine.GET("/info", func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{"appName": appCtx.AppName})
			})
			return engine, engine.Group("/api", appCtx.AuthorizationFilter.Authorize)
		},
		GrpcServer: func(appCtx *ApplicationContext) (*grpc.ServiceDesc, any) {
			return nil, nil
		},
	}
}
