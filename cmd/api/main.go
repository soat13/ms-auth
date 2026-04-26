package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/soat13/oficina-auth/assets/docs"
	"github.com/soat13/oficina-auth/internal/bootstrap"
	"github.com/soat13/oficina-auth/internal/bootstrap/auth"
	"github.com/soat13/oficina-auth/internal/bootstrap/user"
	"github.com/soat13/oficina-utils/pkg/observability"
)

func main() {
	container := bootstrap.BuildDefault()

	obs := observability.Setup(
		container.FiberApp,
		bootstrap.NewDDBPinger(container.DDBClient, container.Cfg.TableName),
	)
	defer observability.Shutdown(obs)
	container.Metrics = obs.Metrics

	docs.Register(container.FiberApp)
	auth.SetupDefault(container)
	user.SetupDefault(container)

	// -----------------------------------------------------------------------------
	// HTTP server start
	// -----------------------------------------------------------------------------

	addr := fmt.Sprintf(":%d", container.Cfg.HTTPPort)
	printUsefulLinks(container.Cfg.HTTPPort)

	if err := container.FiberApp.Listen(addr); err != nil {
		log.Fatal().Err(err).Msg("failed to start HTTP server")
	}
}

func printUsefulLinks(port int) {
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	fmt.Printf("\n🌐 Base URL:        %-45s\n", baseURL)
	fmt.Printf("📚 Swagger/OpenAPI: %-45s\n", baseURL+"/docs")
	fmt.Printf("📊 SonarQube:       %-45s\n", "http://localhost:9000")
}
