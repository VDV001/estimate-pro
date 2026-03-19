package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/daniilrusanov/estimate-pro/backend/internal/config"
	"github.com/daniilrusanov/estimate-pro/backend/internal/infra/postgres"
	infraredis "github.com/daniilrusanov/estimate-pro/backend/internal/infra/redis"
	"github.com/daniilrusanov/estimate-pro/backend/internal/infra/s3"
	"github.com/daniilrusanov/estimate-pro/backend/internal/shared/middleware"
	"github.com/daniilrusanov/estimate-pro/backend/pkg/jwt"

	authHandler "github.com/daniilrusanov/estimate-pro/backend/internal/modules/auth/handler"
	authRepo "github.com/daniilrusanov/estimate-pro/backend/internal/modules/auth/repository"
	authUsecase "github.com/daniilrusanov/estimate-pro/backend/internal/modules/auth/usecase"

	projectDomain "github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/domain"
	projectHandler "github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/handler"
	projectRepo "github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/repository"
	projectUsecase "github.com/daniilrusanov/estimate-pro/backend/internal/modules/project/usecase"

	documentHandler "github.com/daniilrusanov/estimate-pro/backend/internal/modules/document/handler"
	documentRepo "github.com/daniilrusanov/estimate-pro/backend/internal/modules/document/repository"
	documentUsecase "github.com/daniilrusanov/estimate-pro/backend/internal/modules/document/usecase"

	estimationHandler "github.com/daniilrusanov/estimate-pro/backend/internal/modules/estimation/handler"
	estimationRepo "github.com/daniilrusanov/estimate-pro/backend/internal/modules/estimation/repo"
	wsModule "github.com/daniilrusanov/estimate-pro/backend/internal/modules/ws"
	estimationUsecase "github.com/daniilrusanov/estimate-pro/backend/internal/modules/estimation/usecase"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// PostgreSQL
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("connected to postgres")

	// Redis
	rdb, err := infraredis.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	slog.Info("connected to redis")

	// S3 / MinIO
	s3Client, err := s3.NewClient(cfg.S3.Endpoint, cfg.S3.AccessKey, cfg.S3.SecretKey, cfg.S3.Bucket, cfg.S3.UseSSL)
	if err != nil {
		slog.Error("failed to create s3 client", "error", err)
		os.Exit(1)
	}
	if err := s3Client.EnsureBucket(ctx); err != nil {
		slog.Warn("failed to ensure s3 bucket", "error", err)
	}
	slog.Info("s3 client ready")

	// Token store (Redis-backed refresh tokens)
	tokenStore := authRepo.NewRedisTokenStore(rdb)

	// JWT
	jwtService := jwt.NewService(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	// Repositories
	userRepo := authRepo.NewPostgresUserRepository(pool)
	workspaceRepo := projectRepo.NewPostgresWorkspaceRepository(pool)
	projectRepository := projectRepo.NewPostgresProjectRepository(pool)
	memberRepo := projectRepo.NewPostgresMemberRepository(pool)

	// Document repositories
	docRepository := documentRepo.NewPostgresDocumentRepository(pool)
	versionRepo := documentRepo.NewPostgresVersionRepository(pool)
	fileStorage := documentRepo.NewS3FileStorage(s3Client)

	// Estimation repositories
	estRepository := estimationRepo.NewPostgresEstimationRepository(pool)
	estItemRepo := estimationRepo.NewPostgresItemRepository(pool)

	// Usecases
	membershipChecker := authRepo.NewPostgresMembershipChecker(pool)
	authUC := authUsecase.New(userRepo, &workspaceCreatorAdapter{workspaceRepo}, jwtService, tokenStore, &avatarStorageAdapter{s3Client}, membershipChecker)
	projectUC := projectUsecase.New(projectRepository, workspaceRepo, memberRepo)
	documentUC := documentUsecase.New(docRepository, versionRepo, fileStorage)
	estimationUC := estimationUsecase.New(estRepository, estItemRepo)

	// Handlers
	authH := authHandler.New(authUC)
	userFinder := projectRepo.NewUserFinderAdapter(userRepo)
	memberUC := projectUsecase.NewMemberUsecase(memberRepo, projectRepository, userFinder)
	projectH := projectHandler.New(projectUC, memberUC, workspaceRepo)
	documentH := documentHandler.New(documentUC)
	estimationH := estimationHandler.New(estimationUC, &memberRoleAdapter{memberRepo})

	// Router
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS())
	r.Use(chimw.Recoverer)

	// Health check
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// WebSocket hub + event emitter
	wsHub := wsModule.NewHub()
	go wsHub.Run()
	emitEvent := func(eventType, projectID string) {
		wsHub.Broadcast(wsModule.Event{Type: eventType, ProjectID: projectID})
	}
	documentH.SetOnEvent(emitEvent)
	estimationH.SetOnEvent(emitEvent)
	wsH := wsModule.NewHandler(wsHub, jwtService, func(userID string) []string {
		projects, _, _ := projectRepository.ListByUser(context.Background(), userID, 100, 0)
		ids := make([]string, len(projects))
		for i, p := range projects {
			ids[i] = p.ID
		}
		return ids
	})

	// OAuth handler
	oauthH := authHandler.NewOAuthHandler(authUC, cfg.OAuth)

	// Module routes
	authH.Register(r, jwtService)
	oauthH.Register(r)
	wsH.Register(r)
	projectH.Register(r, jwtService)
	documentH.Register(r, jwtService)
	estimationH.Register(r, jwtService)

	// Server
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting server", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	slog.Info("server stopped")
}

// workspaceCreatorAdapter adapts WorkspaceRepository to auth domain's WorkspaceCreator interface.
type workspaceCreatorAdapter struct {
	repo projectDomain.WorkspaceRepository
}

func (a *workspaceCreatorAdapter) CreatePersonalWorkspace(ctx context.Context, userID, name string) error {
	ws := &projectDomain.Workspace{
		ID:        uuid.New().String(),
		Name:      name,
		OwnerID:   userID,
		CreatedAt: time.Now(),
	}
	return a.repo.Create(ctx, ws)
}

// memberRoleAdapter adapts MemberRepository.GetRole to estimation handler's RoleChecker interface.
type memberRoleAdapter struct {
	repo interface {
		GetRole(ctx context.Context, projectID, userID string) (projectDomain.Role, error)
	}
}

func (a *memberRoleAdapter) CanEstimate(ctx context.Context, projectID, userID string) bool {
	role, err := a.repo.GetRole(ctx, projectID, userID)
	if err != nil {
		return true // if role unknown, allow (fail open — permission checked elsewhere)
	}
	return role.CanEstimate()
}

// avatarStorageAdapter adapts S3 Client to auth domain's AvatarStorage interface.
type avatarStorageAdapter struct {
	s3 *s3.Client
}

func (a *avatarStorageAdapter) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	return a.s3.UploadBytes(ctx, key, data, contentType)
}

func (a *avatarStorageAdapter) Download(ctx context.Context, key string) ([]byte, string, error) {
	reader, err := a.s3.Download(ctx, key)
	if err != nil {
		return nil, "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	// Detect content type from data
	ct := http.DetectContentType(data)
	return data, ct, nil
}
