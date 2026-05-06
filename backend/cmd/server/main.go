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
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

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

	estimationDomain "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
	estimationHandler "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/handler"
	estimationRepo "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/repo"
	estimationUsecase "github.com/VDV001/estimate-pro/backend/internal/modules/estimation/usecase"

	extractorDomain "github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
	extractorHandler "github.com/VDV001/estimate-pro/backend/internal/modules/extractor/handler"
	extractorRepo "github.com/VDV001/estimate-pro/backend/internal/modules/extractor/repository"
	extractorUsecase "github.com/VDV001/estimate-pro/backend/internal/modules/extractor/usecase"
	extractorWorker "github.com/VDV001/estimate-pro/backend/internal/modules/extractor/worker"

	reportDomain "github.com/VDV001/estimate-pro/backend/internal/modules/report/domain"
	reportHandler "github.com/VDV001/estimate-pro/backend/internal/modules/report/handler"
	reportUsecase "github.com/VDV001/estimate-pro/backend/internal/modules/report/usecase"

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
	sharedgenerator "github.com/VDV001/estimate-pro/backend/internal/shared/generator"
	sharedllm "github.com/VDV001/estimate-pro/backend/internal/shared/llm"
	sharedreader "github.com/VDV001/estimate-pro/backend/internal/shared/reader"
	sharedsecurity "github.com/VDV001/estimate-pro/backend/internal/shared/security"

	"github.com/VDV001/estimate-pro/backend/internal/infra/composio"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

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
	botUserResolver := botRepo.NewPostgresUserResolver(pool)
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

	// Document generator (PR-B4) — Composite bundles MD / PDF
	// generators, DOCX template filler, and Gotenberg converter.
	// Built unconditionally (small alloc, zero runtime cost) so the
	// generator is ready for the PR-B7 report use case; converter
	// is wired only when GOTENBERG_URL is configured.
	pdfGenerator, err := sharedgenerator.NewPDFGenerator()
	if err != nil {
		slog.Error("failed to construct PDF generator", "error", err)
		os.Exit(1)
	}
	var gotenbergConverter sharedgenerator.Converter
	if cfg.Generator.GotenbergURL != "" {
		gotenbergConverter = sharedgenerator.NewGotenbergConverter(
			cfg.Generator.GotenbergURL,
			cfg.Generator.GotenbergTimeout,
		)
	}
	documentGenerator := sharedgenerator.NewComposite(
		sharedgenerator.NewMDRenderer(),
		pdfGenerator,
		sharedgenerator.NewDOCXRenderer(),
		sharedgenerator.NewDOCXTemplateFiller(),
		gotenbergConverter,
	)
	// Report module — produces PERT estimation reports (md/pdf/docx)
	// from project + aggregated estimation. Wired after the generator
	// is built; the renderer adapter maps reportdomain.Format → the
	// generator's Format and forwards to Composite.Generate.
	reportRendererAdapter := &reportRendererAdapter{generator: documentGenerator}
	reportUC := reportUsecase.NewReporter(
		projectRepository,
		&reportEstimationAggregatorAdapter{estimationUC: estimationUC},
		reportRendererAdapter,
	)
	reportH := reportHandler.New(reportUC)

	// Extractor module (PR-B Document Pipeline) — gated behind the
	// FEATURE_DOCUMENT_PIPELINE_ENABLED flag so a fresh deploy stays
	// dormant until ops opts in. River client is started after the
	// HTTP server is wired (see below); Stop is called on shutdown.
	var extractorH *extractorHandler.Handler
	var riverClient *river.Client[pgx.Tx]
	var botExtraction botDomain.Extractor
	if cfg.Extractor.Enabled {
		extractorRepository := extractorRepo.NewPostgresExtractionRepository(pool)

		readerComposite := sharedreader.NewComposite(
			cfg.Extractor.MaxBytes,
			sharedreader.NewPDFReader(),
			sharedreader.NewDOCXReader(),
			sharedreader.NewMDReader(),
			sharedreader.NewTXTReader(),
			sharedreader.NewCSVReader(),
			sharedreader.NewXLSXReader(),
		)

		extractorCompleter := buildEnvFormatterCompleter(botUsecase.EnvLLMConfig{
			Provider: cfg.LLM.Provider,
			APIKey:   cfg.LLM.APIKey,
			Model:    cfg.LLM.Model,
			BaseURL:  cfg.LLM.BaseURL,
		})

		docSource := &documentSourceAdapter{versionRepo: versionRepo, fileStorage: fileStorage}
		secChecker := &securityCheckerAdapter{}

		extractorWorkerInstance := extractorWorker.NewExtractionWorker(
			extractorRepository,
			docSource,
			readerComposite,
			extractorCompleter,
			secChecker,
		)

		riverWorkers := river.NewWorkers()
		river.AddWorker(riverWorkers, &riverExtractionWorkerAdapter{inner: extractorWorkerInstance})

		var riverErr error
		riverClient, riverErr = river.NewClient(riverpgxv5.New(pool), &river.Config{
			Queues:      map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 5}},
			Workers:     riverWorkers,
			MaxAttempts: 3,
			RetryPolicy: &extractionRetryPolicy{},
		})
		if riverErr != nil {
			slog.Error("failed to create river client", "error", riverErr)
			os.Exit(1)
		}

		enqueuer := &riverJobEnqueuerAdapter{client: riverClient}
		extractorUC := extractorUsecase.NewExtractor(extractorRepository, cfg.Extractor.MaxBytes, enqueuer)
		extractorH = extractorHandler.New(extractorUC)
		botExtraction = &botExtractionAdapter{extractorUC: extractorUC}
		extractorH.WithOwnershipResolver(
			&extractionProjectResolverAdapter{
				extractions: extractorRepository,
				documents:   docRepository,
			},
			&roleGetterAdapter{memberRepo},
		)
	}

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
	envLLM := botUsecase.EnvLLMConfig{
		Provider: cfg.LLM.Provider,
		APIKey:   cfg.LLM.APIKey,
		Model:    cfg.LLM.Model,
		BaseURL:  cfg.LLM.BaseURL,
	}
	botFormatter := botLLM.NewFormatter(buildEnvFormatterCompleter(envLLM))

	botUC := botUsecase.New(
		botSessionRepo,
		botLinkRepo,
		botUserResolver,
		botLLMConfigRepo,
		botMemoryRepo,
		botPrefsRepo,
		botTG,
		botLLM.NewParser,
		envLLM,
		cfg.TelegramBot.BotUsername,
		botFormatter,
		&botProjectAdapter{projectUC: projectUC},
		&botMemberAdapter{memberUC: memberUC},
		&botEstimationAdapter{estimationUC: estimationUC, notifyDispatcher: notifyDispatcher},
		&botDocumentAdapter{documentUC: documentUC},
		&botPasswordResetAdapter{authUC: authUC},
		botExtraction,
		&botReporterAdapter{baseURL: cfg.FrontendBaseURL},
		nil, // textExtractor — wired in PR-issue-8 step 5
		nil, // speechRecognizer — wired in PR-issue-8 step 5
	)
	botH := botHandler.New(botUC, cfg.TelegramBot.WebhookSecret)

	// Module routes
	authRateLimiter := middleware.RateLimit(ctx, 10, time.Minute) // 10 requests per minute per IP
	authH.Register(r, jwtService, authRateLimiter)
	oauthH.Register(r)
	wsH.Register(r)
	membershipMW := middleware.RequireProjectMember(&roleGetterAdapter{memberRepo})
	projectH.Register(r, jwtService, membershipMW)
	documentH.Register(r, jwtService, membershipMW)
	estimationH.Register(r, jwtService, membershipMW)
	notifyH.Register(r, jwtService)
	botH.Register(r)
	if extractorH != nil {
		extractorH.Register(r, jwtService, membershipMW)
	}
	reportH.Register(r, jwtService, membershipMW)

	// Server
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if riverClient != nil {
		if err := riverClient.Start(ctx); err != nil {
			slog.Error("failed to start river client", "error", err)
			os.Exit(1)
		}
		slog.Info("river client started")
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if riverClient != nil {
		if err := riverClient.Stop(shutdownCtx); err != nil {
			slog.Error("river client stop error", "error", err)
		} else {
			slog.Info("river client stopped")
		}
	}

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
	estimationUC     *estimationUsecase.EstimationUsecase
	notifyDispatcher *notifyModule.Dispatcher
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

// SubmitItem creates a single-item estimation for the given task and submits
// it on behalf of the user. Used by the bot `submit_estimation` intent.
//
// Wraps estimation/domain.ErrInvalidHours into bot/domain.ErrInvalidEstimationHours
// so the bot module stays decoupled from estimation/domain's concrete error types.
//
// NOTE: Create+Submit is non-atomic — if Submit fails after Create succeeds,
// the estimation will be left in draft state. Acceptable for low-volume bot
// usage; a proper transactional CreateAndSubmit usecase is a future refactor
// (track separately).
func (a *botEstimationAdapter) SubmitItem(ctx context.Context, projectID, userID, taskName string, minHours, likelyHours, maxHours float64) error {
	input := estimationUsecase.CreateInput{
		ProjectID: projectID,
		UserID:    userID,
		Items: []estimationUsecase.CreateItemInput{
			{TaskName: taskName, MinHours: minHours, LikelyHours: likelyHours, MaxHours: maxHours},
		},
	}
	created, err := a.estimationUC.Create(ctx, input)
	if err != nil {
		if errors.Is(err, estimationDomain.ErrInvalidHours) {
			return botDomain.ErrInvalidEstimationHours
		}
		return fmt.Errorf("botEstimationAdapter.SubmitItem: create: %w", err)
	}
	if err := a.estimationUC.Submit(ctx, created.Estimation.ID, userID); err != nil {
		return fmt.Errorf("botEstimationAdapter.SubmitItem: submit: %w", err)
	}
	return nil
}

// RequestEstimation dispatches an EventEstimationRequested notification to all
// project members except the actor via the notify dispatcher. Used by the bot
// `request_estimation` intent.
func (a *botEstimationAdapter) RequestEstimation(ctx context.Context, projectID, userID, taskName string) error {
	return a.notifyDispatcher.RequestEstimation(ctx, projectID, userID, taskName)
}

// botDocumentAdapter implements botDomain.DocumentManager using existing document usecases.
type botDocumentAdapter struct {
	documentUC *documentUsecase.DocumentUsecase
}

func (a *botDocumentAdapter) Upload(ctx context.Context, projectID, title, fileName string, fileSize int64, fileType string, content io.Reader, userID string) (string, string, error) {
	doc, version, err := a.documentUC.Upload(ctx, documentUsecase.UploadInput{
		ProjectID: projectID,
		Title:     title,
		FileName:  fileName,
		FileSize:  fileSize,
		FileType:  documentDomain.FileType(fileType),
		Content:   content,
		UserID:    userID,
	})
	if err != nil {
		return "", "", err
	}
	return doc.ID, version.ID, nil
}

// botExtractionAdapter implements botDomain.ExtractionTrigger using
// the extractor module's RequestExtraction use case. The bot module
// stays free of cross-module imports — coupling lives here in the
// composition root. nil-safe at the bot side: when the extractor is
// gated off (FEATURE_DOCUMENT_PIPELINE_ENABLED=false), the wiring
// below leaves the adapter nil and the bot short-circuits cleanly.
type botExtractionAdapter struct {
	extractorUC *extractorUsecase.Extractor
}

func (a *botExtractionAdapter) RequestExtraction(ctx context.Context, documentID, documentVersionID string, fileSize int64, actor string) (string, error) {
	ext, err := a.extractorUC.RequestExtraction(ctx, documentID, documentVersionID, fileSize, actor)
	if err != nil {
		return "", err
	}
	return ext.ID, nil
}

// GetExtraction maps the extractor module's domain status / tasks /
// failure_reason onto the bot-side projection so the bot module
// stays free of cross-module imports.
func (a *botExtractionAdapter) GetExtraction(ctx context.Context, extractionID string) (botDomain.ExtractionResult, error) {
	ext, _, err := a.extractorUC.GetExtraction(ctx, extractionID)
	if err != nil {
		return botDomain.ExtractionResult{}, err
	}
	tasks := make([]botDomain.ExtractedTaskSummary, len(ext.Tasks))
	for i, t := range ext.Tasks {
		tasks[i] = botDomain.ExtractedTaskSummary{
			Name:         t.Name,
			EstimateHint: t.EstimateHint,
		}
	}
	return botDomain.ExtractionResult{
		Status:        botDomain.ExtractionStatus(ext.Status),
		Tasks:         tasks,
		FailureReason: ext.FailureReason,
	}, nil
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

// buildEnvFormatterCompleter constructs a shared/llm.Completer from the
// env LLM config for the bot's personality formatter (LLM #2). Composition
// owns the wiring so the bot usecase does not import shared/llm directly.
//
// Returns nil when envLLM is unset or invalid — Formatter handles nil by
// falling back to the raw action result, so bot stays functional even
// without an LLM provider configured. Errors at construction are logged
// once at startup; runtime calls go through the formatter's completer.
func buildEnvFormatterCompleter(envLLM botUsecase.EnvLLMConfig) sharedllm.Completer {
	if envLLM.Provider == "" {
		return nil
	}
	parser, err := sharedllm.NewParser(sharedllm.LLMProviderType(envLLM.Provider), envLLM.APIKey, envLLM.Model, envLLM.BaseURL)
	if err != nil {
		slog.Warn("buildEnvFormatterCompleter: env LLM parser unavailable, formatter will fall back to raw output",
			slog.String("provider", envLLM.Provider),
			slog.String("error", err.Error()))
		return nil
	}
	return parser
}

// ---------------------------------------------------------------------------
// Extractor composition adapters (Document Pipeline, PR-B3)
// All types live here to keep cross-module wiring quarantined to main.go.
// ---------------------------------------------------------------------------

// riverExtractionArgs is the on-wire job type for the extraction queue.
// Kind() returns a stable string so river can match enqueued payloads to
// this worker even across restarts.
type riverExtractionArgs struct {
	ExtractionID string `json:"extraction_id"`
}

func (riverExtractionArgs) Kind() string { return "extraction" }

// riverExtractionWorkerAdapter bridges river.Worker[riverExtractionArgs] to
// the domain-pure ExtractionWorker.Process method. The adapter lives in
// main.go so the worker package stays free of river imports.
type riverExtractionWorkerAdapter struct {
	river.WorkerDefaults[riverExtractionArgs]
	inner *extractorWorker.ExtractionWorker
}

func (a *riverExtractionWorkerAdapter) Work(ctx context.Context, job *river.Job[riverExtractionArgs]) error {
	return a.inner.Process(ctx, extractorWorker.ExtractionArgs{ExtractionID: job.Args.ExtractionID})
}

// riverJobEnqueuerAdapter wraps a river.Client to satisfy usecase.JobEnqueuer.
// InsertOpts sets MaxAttempts to 3 per ADR-016 §retry.
type riverJobEnqueuerAdapter struct {
	client *river.Client[pgx.Tx]
}

func (a *riverJobEnqueuerAdapter) Enqueue(ctx context.Context, extractionID string) error {
	_, err := a.client.Insert(ctx, riverExtractionArgs{ExtractionID: extractionID}, &river.InsertOpts{
		MaxAttempts: 3,
	})
	return err
}

// extractionRetryPolicy implements river.ClientRetryPolicy with the
// fixed schedule from ADR-016 §retry: 5 min → 30 min → 3 hours.
// Delay is selected by the number of prior errors (0-indexed), not the
// attempt counter, to be consistent with river's snooze semantics.
type extractionRetryPolicy struct{}

func (extractionRetryPolicy) NextRetry(job *rivertype.JobRow) time.Time {
	delays := []time.Duration{5 * time.Minute, 30 * time.Minute, 3 * time.Hour}
	idx := len(job.Errors)
	if idx >= len(delays) {
		idx = len(delays) - 1
	}
	return time.Now().UTC().Add(delays[idx])
}

// securityCheckerAdapter wraps shared/security.IsPromptInjection behind the
// worker.SecurityChecker port so the worker package has no direct dependency
// on the shared/security package.
type securityCheckerAdapter struct{}

func (securityCheckerAdapter) IsPromptInjection(text string) bool {
	return sharedsecurity.IsPromptInjection(text)
}

// documentSourceAdapter resolves a DocumentVersionID to its bytes and a
// filename hint by composing the version repository (FileKey + FileType
// lookup) with the S3 file storage client. The extractor module does not
// know S3 keys, bucket names, or document module internals — all coupling
// is quarantined here in the composition root.
type documentSourceAdapter struct {
	versionRepo interface {
		GetByID(ctx context.Context, id string) (*documentDomain.DocumentVersion, error)
	}
	fileStorage interface {
		Download(ctx context.Context, key string) (io.ReadCloser, error)
	}
}

func (a *documentSourceAdapter) Fetch(ctx context.Context, documentVersionID string) ([]byte, string, error) {
	version, err := a.versionRepo.GetByID(ctx, documentVersionID)
	if err != nil {
		return nil, "", fmt.Errorf("documentSource.Fetch: get version %q: %w", documentVersionID, err)
	}
	rc, err := a.fileStorage.Download(ctx, version.FileKey)
	if err != nil {
		return nil, "", fmt.Errorf("documentSource.Fetch: download %q: %w", version.FileKey, err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", fmt.Errorf("documentSource.Fetch: read %q: %w", version.FileKey, err)
	}
	filename := path.Base(version.FileKey)
	if filename == "." || filename == "/" {
		filename = "document." + string(version.FileType)
	}
	return data, filename, nil
}

// extractionProjectResolverAdapter satisfies handler.ExtractionProjectResolver
// by joining the extraction → document → project chain. Cross-module read
// quarantined to main.go; the handler package stays free of cross-module
// imports per Clean Architecture.
type extractionProjectResolverAdapter struct {
	extractions interface {
		GetByID(ctx context.Context, id string) (*extractorDomain.Extraction, error)
	}
	documents interface {
		GetByID(ctx context.Context, id string) (*documentDomain.Document, error)
	}
}

func (a *extractionProjectResolverAdapter) ProjectIDByExtraction(ctx context.Context, extractionID string) (string, error) {
	ext, err := a.extractions.GetByID(ctx, extractionID)
	if err != nil {
		return "", fmt.Errorf("extractionProjectResolver: get extraction %q: %w", extractionID, err)
	}
	doc, err := a.documents.GetByID(ctx, ext.DocumentID)
	if err != nil {
		return "", fmt.Errorf("extractionProjectResolver: get document %q: %w", ext.DocumentID, err)
	}
	return doc.ProjectID, nil
}

// reportRendererAdapter satisfies report/usecase.Renderer by mapping
// reportdomain.Format → shared/generator.Format and forwarding to
// Composite.Generate. Centralises the format-mapping concern in the
// composition root so the report use case stays decoupled from the
// generator package.
type reportRendererAdapter struct {
	generator *sharedgenerator.Composite
}

func (a *reportRendererAdapter) Render(ctx context.Context, format reportDomain.Format, input sharedgenerator.GenerationInput) ([]byte, error) {
	switch format {
	case reportDomain.FormatMD:
		return a.generator.Generate(ctx, sharedgenerator.FormatMD, input)
	case reportDomain.FormatPDF:
		return a.generator.Generate(ctx, sharedgenerator.FormatPDF, input)
	case reportDomain.FormatDOCX:
		return a.generator.Generate(ctx, sharedgenerator.FormatDOCX, input)
	}
	return nil, fmt.Errorf("reportRendererAdapter: unsupported format %q", format)
}

// reportEstimationAggregatorAdapter satisfies report/usecase.EstimationAggregator
// by forwarding to the existing estimation use case. Keeps the report module
// from importing estimation/usecase directly — only the composition root
// crosses module boundaries.
type reportEstimationAggregatorAdapter struct {
	estimationUC *estimationUsecase.EstimationUsecase
}

func (a *reportEstimationAggregatorAdapter) GetAggregated(ctx context.Context, projectID string) (*estimationDomain.AggregatedResult, error) {
	return a.estimationUC.GetAggregated(ctx, projectID)
}


// botReporterAdapter satisfies bot/domain.Reporter by building a
// deeplink to the frontend's existing report download path. Bot
// users get a clickable link in chat instead of a direct file
// transfer — the telegram client API is intentionally kept
// minimal (text only) for this PR.
type botReporterAdapter struct {
	baseURL string
}

func (a *botReporterAdapter) BuildReportURL(_ context.Context, projectID, format string) (string, error) {
	if format == "" {
		format = "pdf"
	}
	return fmt.Sprintf("%s/dashboard/projects/%s?download=report&format=%s", a.baseURL, projectID, format), nil
}
