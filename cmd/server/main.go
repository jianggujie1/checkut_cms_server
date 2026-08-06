package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"checkut-cms-server/internal/config"
	"checkut-cms-server/internal/controller"
	"checkut-cms-server/internal/repository"
	"checkut-cms-server/internal/service"
	"checkut-cms-server/internal/supabase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := repository.NewPool(ctx, cfg.CMSDBDSN)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	supa := supabase.New(cfg.SupabaseURL, cfg.SupabaseKey, cfg.StorageBucket)

	// repositories
	destRepo := repository.NewDestinationRepo(pool)
	attrRepo := repository.NewAttractionRepo(pool)
	itRepo := repository.NewItineraryRepo(pool)
	metaRepo := repository.NewPublishMetaRepo(pool)
	pubRepo := repository.NewPublishRepo(pool)

	// services
	destSvc := service.NewDestinationService(destRepo)
	attrSvc := service.NewAttractionService(attrRepo)
	itSvc := service.NewItineraryService(itRepo)
	uploadSvc := service.NewUploadService(supa)
	syncSvc := service.NewSyncService(pubRepo, metaRepo, supa)
	pubSvc := service.NewPublishService(pubRepo, metaRepo, supa)

	// controllers
	destCtrl := controller.NewDestinationController(destSvc)
	attrCtrl := controller.NewAttractionController(attrSvc)
	itCtrl := controller.NewItineraryController(itSvc)
	uploadCtrl := controller.NewUploadController(uploadSvc)
	syncCtrl := controller.NewSyncController(syncSvc)
	pubCtrl := controller.NewPublishController(pubSvc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/destinations", func(sub chi.Router) {
			sub.Get("/", destCtrl.List)
			sub.Post("/", destCtrl.Create)
			sub.Get("/{id}", destCtrl.Get)
			sub.Put("/{id}", destCtrl.Update)
			sub.Patch("/{id}/status", destCtrl.SetStatus)
			sub.Delete("/{id}", destCtrl.Delete)
		})
		api.Route("/attractions", func(sub chi.Router) {
			sub.Get("/", attrCtrl.List)
			sub.Post("/", attrCtrl.Create)
			sub.Get("/{id}", attrCtrl.Get)
			sub.Put("/{id}", attrCtrl.Update)
			sub.Patch("/{id}/status", attrCtrl.SetStatus)
			sub.Delete("/{id}", attrCtrl.Delete)
		})
		api.Route("/itineraries", func(sub chi.Router) {
			sub.Get("/", itCtrl.List)
			sub.Post("/", itCtrl.Create)
			sub.Get("/{id}", itCtrl.Get)
			sub.Put("/{id}", itCtrl.Update)
			sub.Patch("/{id}/status", itCtrl.SetStatus)
			sub.Delete("/{id}", itCtrl.Delete)
		})
		api.Post("/uploads", uploadCtrl.Upload)
		api.Route("/sync", func(sub chi.Router) {
			sub.Get("/status", syncCtrl.Status)
			sub.Post("/import", syncCtrl.Import)
		})
		api.Route("/publish", func(sub chi.Router) {
			sub.Get("/diff", pubCtrl.Diff)
			sub.Post("/", pubCtrl.Run)
		})
	})

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: r,
	}

	go func() {
		log.Printf("Checkut CMS listening on http://%s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
