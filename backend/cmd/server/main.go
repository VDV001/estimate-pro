// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/VDV001/estimate-pro/backend/internal/config"
	"github.com/VDV001/estimate-pro/backend/internal/infra/postgres"
	infraredis "github.com/VDV001/estimate-pro/backend/internal/infra/redis"
	"github.com/VDV001/estimate-pro/backend/internal/infra/s3"
	"github.com/VDV001/estimate-pro/backend/internal/shared/middleware"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"

	authDomain "github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
	authHandler "github.com/VDV001/estimate-pro/backend/internal/modules/auth/handler"
	authRepo "github.com/VDV001/estimate-pro/backend/internal/modules/auth/repository"
	authUsecase "github.com/VDV001/estimate-pro/backend/internal/modules/auth/usecase"

	projectDomain "github.com/VDV001/estimate-pro/backend/internal/modules/project/domain"
	projectHandler "github.com/VDV001/estimate-pro/backend/internal/modules/project/handler"
	projectRepo "github.com/VDV001/estimate-pro/backend/internal/modules/project/repository"
	projectUsecase "github.com/VDV001/estimate-pro/backend/internal/modules/project/usecase"

	documentDomain "github.com/VDV001/estimate-pro/backend/internal/modules/document/domain"
	documentHandler "github.com/VDV001/estimate-pro/backend/internal/modules/document/handler"
	documentRepo "github.com/VDV001/estimate-pro/backend/internal/modules/document/repository"
	documentUsecase "github.com/VDV001/estimate-pro/backend/internal/modules/document/usecase"

	estimationHandler "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/handler"
	estimationRepo "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/repo"
	estimationUsecase "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/usecase"

	notifyModule "github.com/VDV001/estimate-pro/backend/internal/modules/notify"
	notifyChannel "github.com/VDV001/estimate-pro/backend/internal/modules/notify/channel"
	notifyHandler "github.com/VDV001/estimate-pro/backend/internal/modules/notify/handler"
	notifyRepo "github.com/VDV001/estimate-pro/backend/internal/modules/notify/repository"
	notifyUsecase "github.com/VDV001/estimate-pro/backend/internal/modules/notify/usecase"

	wsModule "github.com/VDV001/estimate-pro/backend/internal/modules/ws"

	botDomain "github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
	botHandler "github.com/VDV001/estimate-pro/backend/internal/modules/bot/handler"
	botLLM "github.com/VDV001/estimate-pro/backend/internal/modules/bot/llm"
	botRepo "github.com/VDV001/estimate-pro/backend/internal/modules/bot/repository"
	botTelegram "github.com/VDV001/estimate-pro/backend/internal/modules/bot/telegram"
	botUsecase "github.com/VDV001/estimate-pro/backend/internal/modules/bot/usecase"

	"github.com/VDV001/estimate-pro/backend/internal/infra/composio"
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

	// Notification repositories
	notifRepository := notifyRepo.NewPostgresNotificationRepository(pool)
	prefRepository := notifyRepo.NewPostgresPreferenceRepository(pool)
	deliveryLogRepo := notifyRepo.NewPostgresDeliveryLogRepository(pool)

	// Bot module repositories
	botSessionRepo := botRepo.NewPostgresSessionRepository(pool)
	botLinkRepo := botRepo.NewPostgresUserLinkRepository(pool)
	botLLMConfigRepo := botRepo.NewPostgresLLMConfigRepository(pool)
	botMemoryRepo := botRepo.NewPostgresMemoryRepository(pool)
	botPrefsRepo := botRepo.NewPostgresUserPrefsRepository(pool)

	// Bot Telegram client
	botTG := botTelegram.NewClient(cfg.TelegramBot.Token)

	// Composio client + external senders
	composioClient := composio.NewClient(cfg.Composio.APIKey)
	emailLookup := notifyRepo.NewEmailLookup(pool)
	emailSender := notifyChannel.NewEmailSender(composioClient, cfg.Composio.GmailAccountID, emailLookup)
	telegramLookup := notifyRepo.NewTelegramChatLookup(pool)
	telegramSender := notifyChannel.NewTelegramSender(composioClient, cfg.Composio.TelegramAccountID, telegramLookup)
	memberLister := notifyRepo.NewMemberListerAdapter(pool)

	// Usecases
	membershipChecker := authRepo.NewPostgresMembershipChecker(pool)
	authUC := authUsecase.New(userRepo, &workspaceCreatorAdapter{workspaceRepo}, jwtService, tokenStore, &avatarStorageAdapter{s3Client}, membershipChecker)

	// Password reset wiring
	resetTokenStore := authRepo.NewRedisResetTokenStore(rdb)
	resetNotifier := &resetNotifierAdapter{
		userRepo:       userRepo,
		emailSender:    emailSender,
		telegramClient: botTG,
	}
	authUC.SetResetConfig(resetTokenStore, cfg.FrontendBaseURL, 15*time.Minute)
	authUC.SetResetNotifier(resetNotifier)

	projectUC := projectUsecase.New(projectRepository, workspaceRepo, memberRepo)
	documentUC := documentUsecase.New(docRepository, versionRepo, fileStorage)
	estimationUC := estimationUsecase.New(estRepository, estItemRepo)
	notifyUC := notifyUsecase.New(notifRepository, prefRepository, deliveryLogRepo, memberLister, emailSender, telegramSender)

	// Handlers
	authH := authHandler.New(ctx, authUC)
	memberUC := projectUsecase.NewMemberUsecase(memberRepo, projectRepository, &userFinderAdapter{repo: userRepo})
	workspaceUC := projectUsecase.NewWorkspaceUsecase(workspaceRepo)
	projectH := projectHandler.New(projectUC, memberUC, workspaceUC)
	documentH := documentHandler.New(documentUC)
	estimationH := estimationHandler.New(estimationUC, &memberRoleAdapter{memberRepo})

	// Router
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins...))
	r.Use(middleware.MaxBodySize(50 << 20)) // 50MB max request body
	r.Use(chimw.Recoverer)

	// Health check
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Notification dispatcher
	userNameLookup := notifyRepo.NewUserNameLookup(pool)
	notifyDispatcher := notifyModule.NewDispatcher(notifyUC, userNameLookup, ctx)

	// WebSocket hub + event emitter
	wsHub := wsModule.NewHub()
	go wsHub.Run()
	emitEvent := func(eventType, projectID, userID string) {
		wsHub.Broadcast(wsModule.Event{Type: eventType, ProjectID: projectID, UserID: userID})
		notifyDispatcher.HandleEvent(eventType, projectID, userID)
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

	// Notification handler
	notifyH := notifyHandler.New(notifyUC)

	// Bot module
	botUC := botUsecase.New(
		botSessionRepo,
		botLinkRepo,
		botLLMConfigRepo,
		botMemoryRepo,
		botPrefsRepo,
		botTG,
		botLLM.NewParser,
		botUsecase.EnvLLMConfig{
			Provider: cfg.LLM.Provider,
			APIKey:   cfg.LLM.APIKey,
			Model:    cfg.LLM.Model,
			BaseURL:  cfg.LLM.BaseURL,
		},
		cfg.TelegramBot.BotUsername,
		&botProjectAdapter{projectUC: projectUC},
		&botMemberAdapter{memberUC: memberUC},
		&botEstimationAdapter{estimationUC: estimationUC},
		&botDocumentAdapter{documentUC: documentUC},
		&botPasswordResetAdapter{authUC: authUC},
	)
	botH := botHandler.New(botUC, cfg.TelegramBot.WebhookSecret)

	// Module routes
	authRateLimiter := middleware.RateLimit(10, time.Minute) // 10 requests per minute per IP
	authH.Register(r, jwtService, authRateLimiter)
	oauthH.Register(r)
	wsH.Register(r)
	membershipMW := middleware.RequireProjectMember(&roleGetterAdapter{memberRepo})
	projectH.Register(r, jwtService, membershipMW)
	documentH.Register(r, jwtService, membershipMW)
	estimationH.Register(r, jwtService, membershipMW)
	notifyH.Register(r, jwtService)
	botH.Register(r)

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
		return false // fail-closed: deny on error
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

// botProjectAdapter implements botDomain.ProjectManager using existing project usecases.
type botProjectAdapter struct {
	projectUC *projectUsecase.ProjectUsecase
}

func (a *botProjectAdapter) Create(ctx context.Context, workspaceID, name, description, userID string) (string, error) {
	project, err := a.projectUC.Create(ctx, projectUsecase.CreateProjectInput{
		WorkspaceID: workspaceID,
		Name:        name,
		Description: description,
		UserID:      userID,
	})
	if err != nil {
		return "", err
	}
	return project.ID, nil
}

func (a *botProjectAdapter) Update(ctx context.Context, projectID, name, description, userID string) error {
	_, err := a.projectUC.Update(ctx, projectUsecase.UpdateProjectInput{
		ID:          projectID,
		Name:        name,
		Description: description,
		UserID:      userID,
	})
	return err
}

func (a *botProjectAdapter) ListByUser(ctx context.Context, userID string, limit, offset int) ([]botDomain.ProjectSummary, int, error) {
	result, err := a.projectUC.ListByUser(ctx, projectUsecase.ListByUserInput{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, err
	}
	summaries := make([]botDomain.ProjectSummary, len(result.Projects))
	for i, p := range result.Projects {
		summaries[i] = botDomain.ProjectSummary{
			ID:     p.ID,
			Name:   p.Name,
			Status: string(p.Status),
		}
	}
	return summaries, result.Total, nil
}

// botMemberAdapter implements botDomain.MemberManager using existing member usecases.
type botMemberAdapter struct {
	memberUC *projectUsecase.MemberUsecase
}

func (a *botMemberAdapter) AddByEmail(ctx context.Context, projectID, email, role, callerID string) error {
	return a.memberUC.AddMemberByEmail(ctx, projectUsecase.AddMemberByEmailInput{
		ProjectID: projectID,
		Email:     email,
		Role:      projectDomain.Role(role),
		CallerID:  callerID,
	})
}

func (a *botMemberAdapter) Remove(ctx context.Context, projectID, userID, callerID string) error {
	return a.memberUC.RemoveMember(ctx, projectUsecase.RemoveMemberInput{
		ProjectID: projectID,
		UserID:    userID,
		CallerID:  callerID,
	})
}

func (a *botMemberAdapter) List(ctx context.Context, projectID string) ([]botDomain.MemberSummary, error) {
	members, err := a.memberUC.ListMembersWithUsers(ctx, projectID)
	if err != nil {
		return nil, err
	}
	summaries := make([]botDomain.MemberSummary, len(members))
	for i, m := range members {
		summaries[i] = botDomain.MemberSummary{
			UserID:    m.UserID,
			UserName:  m.UserName,
			UserEmail: m.UserEmail,
			Role:      string(m.Role),
		}
	}
	return summaries, nil
}

// botEstimationAdapter implements botDomain.EstimationManager using existing estimation usecases.
type botEstimationAdapter struct {
	estimationUC *estimationUsecase.EstimationUsecase
}

func (a *botEstimationAdapter) GetAggregated(ctx context.Context, projectID string) (string, error) {
	result, err := a.estimationUC.GetAggregated(ctx, projectID)
	if err != nil {
		return "", err
	}
	if len(result.Items) == 0 {
		return "No estimations yet", nil
	}
	var sb strings.Builder
	for _, item := range result.Items {
		sb.WriteString(fmt.Sprintf("• %s: %.1f ч\n", item.TaskName, item.AvgPERTHours))
	}
	sb.WriteString(fmt.Sprintf("\nИтого: %.1f ч", result.TotalHours))
	return sb.String(), nil
}

// botDocumentAdapter implements botDomain.DocumentManager using existing document usecases.
type botDocumentAdapter struct {
	documentUC *documentUsecase.DocumentUsecase
}

func (a *botDocumentAdapter) Upload(ctx context.Context, projectID, title, fileName string, fileSize int64, fileType string, content io.Reader, userID string) error {
	_, _, err := a.documentUC.Upload(ctx, documentUsecase.UploadInput{
		ProjectID: projectID,
		Title:     title,
		FileName:  fileName,
		FileSize:  fileSize,
		FileType:  documentDomain.FileType(fileType),
		Content:   content,
		UserID:    userID,
	})
	return err
}

// roleGetterAdapter adapts MemberRepository.GetRole to middleware.RoleGetter interface (returns string).
type roleGetterAdapter struct {
	repo interface {
		GetRole(ctx context.Context, projectID, userID string) (projectDomain.Role, error)
	}
}

func (a *roleGetterAdapter) GetRole(ctx context.Context, projectID, userID string) (string, error) {
	role, err := a.repo.GetRole(ctx, projectID, userID)
	if err != nil {
		return "", err
	}
	return string(role), nil
}

// userFinderAdapter adapts auth UserRepository to project.domain.UserFinder.
// Lives in main.go to avoid cross-module import between project and auth.
type userFinderAdapter struct {
	repo *authRepo.PostgresUserRepository
}

func (a *userFinderAdapter) FindByEmail(ctx context.Context, email string) (string, error) {
	user, err := a.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("userFinder.FindByEmail: %w", err)
	}
	return user.ID, nil
}

// resetNotifierAdapter delivers password-reset links via email and Telegram.
type resetNotifierAdapter struct {
	userRepo interface {
		GetByID(ctx context.Context, id string) (*authDomain.User, error)
	}
	emailSender interface {
		Send(ctx context.Context, userID, subject, body string) error
	}
	telegramClient interface {
		SendMarkdown(ctx context.Context, chatID, text string) error
	}
}

func (a *resetNotifierAdapter) NotifyReset(ctx context.Context, userID, resetLink string) error {
	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("resetNotifier: get user: %w", err)
	}

	// Email delivery
	subject := "EstimatePro — Password Reset"
	body := fmt.Sprintf("Click the link to reset your password:\n%s\n\nLink expires in 15 minutes.", resetLink)
	if err := a.emailSender.Send(ctx, userID, subject, body); err != nil {
		slog.Warn("reset email failed", "user_id", userID, "error", err)
	}

	// Telegram delivery (if user has linked Telegram via bot)
	if user.TelegramChatID != "" {
		tgMsg := fmt.Sprintf("*EstimatePro — Сброс пароля*\n\nСсылка для сброса пароля:\n%s\n\n_Ссылка действительна 15 минут._", resetLink)
		if err := a.telegramClient.SendMarkdown(ctx, user.TelegramChatID, tgMsg); err != nil {
			slog.Warn("reset telegram failed", "user_id", userID, "error", err)
		}
	}

	return nil
}

// botPasswordResetAdapter implements botDomain.PasswordResetManager using existing auth usecase.
type botPasswordResetAdapter struct {
	authUC *authUsecase.AuthUsecase
}

func (a *botPasswordResetAdapter) RequestReset(ctx context.Context, userID string) (string, error) {
	link, err := a.authUC.ForgotPasswordByUserID(ctx, userID)
	if err != nil {
		// Translate auth domain error to bot domain error (no cross-module import).
		if errors.Is(err, authDomain.ErrNoPassword) {
			return "", botDomain.ErrNoPassword
		}
		return "", err
	}
	return link, nil
}
