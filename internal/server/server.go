package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wordsGo/internal/config"
	"wordsGo/internal/handlers"
	"wordsGo/internal/repository"
	"wordsGo/internal/service"
	"wordsGo/slogger"
)

func Run(cfg config.Config) {

	db, err := repository.NewPostgres(cfg.DB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewUserRepo(db)
	dictionaryRepo := repository.NewDictionaryRepo(db)

	userService := service.NewUserService(repo, cfg.JWT.Secret, cfg.JWT.TTL)
	dictionaryService := service.NewDictionaryService(dictionaryRepo)

	ctxBG := context.Background()
	if err := userService.SyncAdmin(ctxBG, cfg.Admin); err != nil {
		log.Fatal("Failed to sync admin user:", err)
	}

	dictPath := os.Getenv("DICTIONARY_PATH")
	if dictPath == "" {
		dictPath = "eng-rus_Google_v4.json"
	}

	if err := dictionaryService.LoadDictionary(ctxBG, dictPath); err != nil {
		log.Printf("Failed to load dictionary: %v", err)
		// Решите, нужно ли падать (log.Fatal) или просто логировать ошибку
	}

	userHandler := handlers.NewHandler(userService)

	router := handlers.RegisterRoutes(userHandler, cfg.JWT.Secret)

	srv := &http.Server{
		Addr:         ":" + cfg.Api.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slogger.Log.Info("Listening on port", "port", cfg.Api.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting gracefully")

}
