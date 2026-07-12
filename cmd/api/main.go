package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"omie-sync-api/internal/audit"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/config"
	"omie-sync-api/internal/dados"
	"omie-sync-api/internal/db"
	"omie-sync-api/internal/empresas"
	"omie-sync-api/internal/etl"
	"omie-sync-api/internal/etl/progress"
	"omie-sync-api/internal/grupos"
	"omie-sync-api/internal/logger"
	"omie-sync-api/internal/omie_config"
	"omie-sync-api/internal/permissoes"
	"omie-sync-api/internal/query"
	"omie-sync-api/internal/server"
	syncsvc "omie-sync-api/internal/sync"
	"omie-sync-api/internal/usuarios"
	"omie-sync-api/internal/webhooks"
	"omie-sync-api/internal/worker"
)

func main() {
	// --- Config ---
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	// --- Logger ---
	log := logger.New(cfg.AppEnv)
	log.Info().Str("env", cfg.AppEnv).Msg("iniciando omie-sync-api")

	// --- DB Pool ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPoolWithConcurrency(ctx, cfg.DatabaseURL, cfg.WorkerMaxConcurrent)
	if err != nil {
		log.Fatal().Err(err).Msg("falha ao conectar ao banco")
	}
	defer pool.Close()
	log.Info().Msg("conexão com banco estabelecida")

	// --- Repositories ---
	auditRepo := audit.NewRepository(pool)
	authRepo := auth.NewRepository(pool)
	gruposRepo := grupos.NewRepository(pool)
	empresasRepo := empresas.NewRepository(pool)
	syncRepo := syncsvc.NewRepository(pool)
	webhooksRepo := webhooks.NewRepository(pool)
	usuariosRepo := usuarios.NewRepository(pool)
	permissoesRepo := permissoes.NewRepository(pool)
	omieConfigRepo := omie_config.NewRepository(pool)

	// --- SSE Hub ---
	sseHub := syncsvc.NewSSEHub()

	// --- Provisioner ---
	provisioner := db.NewProvisioner(pool)

	// --- Services ---
	jwtSvc := auth.NewJWTService(cfg.JWTSecret)

	authSvc := auth.NewService(authRepo, jwtSvc)
	gruposSvc := grupos.NewService(gruposRepo, provisioner)
	empresasSvc := empresas.NewService(empresasRepo)
	dispatcher := webhooks.NewDispatcher(webhooksRepo, log)
	syncSvc := syncsvc.NewService(syncRepo, dispatcher, log)
	usuariosSvc := usuarios.NewService(usuariosRepo)
	permissoesSvc := permissoes.NewService(permissoesRepo)
	omieConfigSvc := omie_config.NewService(omieConfigRepo)

	// --- Handlers ---
	authHandler := auth.NewHandler(authSvc, jwtSvc)
	gruposHandler := grupos.NewHandler(gruposSvc, jwtSvc)
	empresasHandler := empresas.NewHandler(empresasSvc, jwtSvc)
	syncHandler := syncsvc.NewHandler(syncSvc, jwtSvc, sseHub)
	usuariosHandler := usuarios.NewHandler(usuariosSvc, jwtSvc)
	permissoesHandler := permissoes.NewHandler(permissoesSvc, jwtSvc)
	dadosHandler := dados.NewHandler(pool, jwtSvc)
	omieConfigHandler := omie_config.NewHandler(omieConfigSvc, jwtSvc)
	querySvc := query.NewService()
	queryHandler := query.NewHandler(querySvc, pool, jwtSvc)

	// --- ETL Worker + Scheduler ---
	executors := etl.NewAllExecutors(pool, log)
	fetcher := worker.NewEmpresaFetcher(pool)
	reporter := progress.NewDBReporter(pool, syncRepo)
	etlWorker := worker.NewWorker(syncRepo, fetcher, executors, dispatcher, reporter, omieConfigSvc, sseHub, pool, log)
	scheduler := worker.NewScheduler(pool, syncRepo, etlWorker, cfg.WorkerMaxConcurrent, log)

	// Registra o worker e o pool de concorrência no syncSvc
	syncsvc.SetProcessor(syncSvc, etlWorker)
	syncsvc.SetSubmitter(syncSvc, scheduler.Pool())

	// Recupera jobs que ficaram presos em caso de reinício inesperado
	if err := syncSvc.StartupRecovery(context.Background()); err != nil {
		log.Error().Err(err).Msg("falha no startup recovery — continuando mesmo assim")
	}

	scheduler.Start()
	defer scheduler.Stop()

	// --- Background Jobs ---
	deletionJob := empresas.NewDeletionJob(empresasRepo, log)
	deletionJob.Start()
	defer deletionJob.Stop()

	// --- Router ---
	router := server.NewRouter(server.Dependencies{
		AuditRepo:         auditRepo,
		AuthHandler:       authHandler,
		GruposHandler:     gruposHandler,
		EmpresasHandler:   empresasHandler,
		SyncHandler:       syncHandler,
		UsuariosHandler:   usuariosHandler,
		PermissoesHandler: permissoesHandler,
		DadosHandler:      dadosHandler,
		OmieConfigHandler: omieConfigHandler,
		QueryHandler:      queryHandler,
		Logger:            log,
	})

	// --- HTTP Server ---
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan struct{})
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit
		log.Info().Str("signal", sig.String()).Msg("sinal de shutdown recebido")

		shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutCancel()

		if err := srv.Shutdown(shutCtx); err != nil {
			log.Error().Err(err).Msg("erro no shutdown do servidor")
		}
		close(done)
	}()

	log.Info().Str("port", cfg.Port).Msg("servidor iniciado")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("erro ao iniciar servidor")
	}

	<-done
	log.Info().Msg("servidor encerrado")
}
