package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	httpSwagger "github.com/swaggo/http-swagger"

	"wordsGo/internal/middleware"
	"wordsGo/internal/models"
	"wordsGo/internal/service"
	"wordsGo/internal/utils"
	"wordsGo/slogger"
)

type Handler struct {
	userService service.UserService
	dict        service.DictionaryService
}

const ctxWithTimeout time.Duration = time.Second * 5

func NewHandler(us service.UserService, ds service.DictionaryService) *Handler {
	return &Handler{
		userService: us,
		dict:        ds,
	}
}

func RegisterRoutes(h *Handler, jwtSecret string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.LoggerMiddleware)

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1/wordsGo", func(r chi.Router) {

		r.Post("/users", h.CreateUser)
		r.Post("/login", h.Login)
		r.Get("/users/{id}", h.GetUserByID)
		r.Get("/users", h.GetUsers)

		r.Group(func(r chi.Router) {
			//r.Use(middleware.BasicAuthMiddleware(h.userService.Authenticate))
			r.Use(middleware.AuthMidleware(jwtSecret))

			r.Put("/users/{id}", h.UpdateUser)
			r.Delete("/users/{id}", h.DeleteUser)
			r.Get("/words", h.GetWords)
			r.Post("/addWords", h.AddWords)

			r.Get("/dictionary/search", h.SearchDictionary) // Поиск: GET /api/v1/wordsGo/dictionary/search?q=hel
			r.Post("/users/words", h.AddWordToLearning)     // Добавление: POST /api/v1/wordsGo/users/words body:{"word_id":"..."}

		})
	})
	return r
}

// Login
// @Summary User Login
// @Description Login and get JWT token
// @Tags users
// @Accept json
// @Produce json
// @Param input body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
// @Failure 401 {object} handlers.JSONError "Invalid credentials"
// @Router /login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()

	token, err := h.userService.Login(ctx, req)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			WriteError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		slogger.Log.ErrorContext(ctx, "Login failed", "err", err)
		return
	}

	JSONResponse(w, http.StatusOK, models.LoginResponse{Token: token})
}

// CreateUser creates a new user
// @Summary CreateUser a new user
// @Description Register a new user in the system
// @Tags users
// @Accept json
// @Produce json
// @Param input body models.CreateUserRequest true "User registration info"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} handlers.JSONError "Invalid input or validation info"
// @Failure 409 {object} handlers.JSONError "User already exists"
// @Failure 500 {object} handlers.JSONError "Internal server error"
// @Router /users [post]
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {

	var req models.CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		slogger.Log.ErrorContext(r.Context(), "Invalid request body", "err", err)
		return
	}

	if req.Email == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" {
		WriteError(w, http.StatusBadRequest, "Fields cannot be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()

	slogger.Log.DebugContext(ctx, "Creating user",
		"email", req.Email,
		"firstName", req.FirstName,
		"lastName", req.LastName,
	)

	createdUser, err := h.userService.Create(ctx, req)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrUserAlreadyExists):
			WriteError(w, http.StatusConflict, "User already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		slogger.Log.ErrorContext(ctx, "Failed to create user", "err", err)
		return
	}

	JSONResponse(w, http.StatusCreated, createdUser)
}

// GetUserByID Get User By ID
// @Summary Get user by ID
// @Description Get User By ID from DB
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User UUID" format(uuid)
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} handlers.JSONError "Invalid UUID"
// @Failure 404 {object} handlers.JSONError "User not found"
// @Failure 500 {object} handlers.JSONError "Internal server error"
// @Router /users/{id} [get]
func (h *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()

	user, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrUserNotFound):
			WriteError(w, http.StatusNotFound, "User not found")
			slogger.Log.DebugContext(ctx, "User not found", "err", err, "id", id)
			return
		default:
			WriteError(w, http.StatusInternalServerError, "Failed to get user")
			slogger.Log.ErrorContext(ctx, "Failed to get user", "err", err)
			return
		}
	}
	JSONResponse(w, http.StatusOK, user)
}

// GetUsers Get list of users
// @Summary Get list of users
// @Description Get list of all users from DB
// @Tags users
// @Accept json
// @Produce json
// @Param limit query int false "Limit records per page (default 10)"
// @Param page query int false "Page number (default 1)"
// @Param order query string false "Sort order: asc or desc (default desc)"
// @Success 200 {object} models.ListOfUsersResponse
// @Failure 500 {object} handlers.JSONError "Failed to get users"
// @Router /users [get]
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {

	var (
		limit uint64 = 10
		page  uint64 = 1
	)

	if s := r.URL.Query().Get("limit"); s != "" {
		l, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid limit")
			return
		}
		limit = l
	}

	if s := r.URL.Query().Get("page"); s != "" {
		p, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid page")
			return
		}
		page = p
	}

	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}

	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()

	users, err := h.userService.GetUsers(ctx, limit, page, order)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get users")
		slogger.Log.ErrorContext(ctx, "Failed to get users", "err", err)
		return
	}
	JSONResponse(w, http.StatusOK, users)
}

// DeleteUser User delete by ID
// @Summary DeleteUser user by ID
// @Description DeleteUser user by ID from DB
// @Tags users
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param id path string true "User UUID" format(uuid)
// @Success 200 {object} nil
// @Failure 400 {object} handlers.JSONError "Invalid user ID"
// @Failure 403 {object} handlers.JSONError "You can delete only your own account"
// @Failure 500 {object} handlers.JSONError "Failed to delete user"
// @Router /users/{id} [delete]
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	requester := r.Context().Value(middleware.UserCtxKey{}).(*models.User)

	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()

	err = h.userService.Delete(ctx, requester, targetID)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrUserNotFound):
			WriteError(w, http.StatusNotFound, "User not found")
			return
		case errors.Is(err, models.ErrPermissionDenied):
			WriteError(w, http.StatusForbidden, "Permission denied")
			return

		}
		WriteError(w, http.StatusInternalServerError, "Failed to delete user")
		slogger.Log.ErrorContext(ctx, "Failed to delete user", "err", err)
		return
	}
	JSONResponse(w, http.StatusOK, nil)
}

// UpdateUser User update by ID
// @Summary UpdateUser user by ID
// @Description UpdateUser user by ID into DB
// @Tags users
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param id path string true "User UUID" format(uuid)
// @Param input body models.UpdateUserRequest true "User update info"
// @Success 200 {object} models.UserResponse "Successfully updated"
// @Failure 400 {object} handlers.JSONError "Invalid user ID"
// @Failure 403 {object} handlers.JSONError "You can delete only your own account"
// @Failure 500 {object} handlers.JSONError "Failed to delete user"
// @Router /users/{id} [put]
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	authUser := r.Context().Value(middleware.UserCtxKey{}).(*models.User)

	slogger.Log.DebugContext(r.Context(), "Updating user", "id", id, "authUser", authUser)

	if id != authUser.ID {
		WriteError(w, http.StatusForbidden, "You can update only your own account")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		slogger.Log.ErrorContext(r.Context(), "Invalid request body", "err", err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()

	if req.Password != nil {
		if err := utils.ValidatePassword(*req.Password); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	updatedUser, err := h.userService.Update(ctx, authUser.ID, req)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrUserNotFound):
			WriteError(w, http.StatusNotFound, "User not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to update user")
		slogger.Log.ErrorContext(ctx, "Failed to update user", "err", err)
		return
	}

	JSONResponse(w, http.StatusOK, updatedUser)
}

func (h *Handler) GetWords(w http.ResponseWriter, r *http.Request) {
	// 1. Извлекаем юзера из контекста
	user, ok := r.Context().Value(middleware.UserCtxKey{}).(*models.User)
	if !ok {
		WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var (
		limit uint64 = 10
		page  uint64 = 1
	)

	if s := r.URL.Query().Get("limit"); s != "" {
		l, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid limit")
			return
		}
		limit = l
	}

	if s := r.URL.Query().Get("page"); s != "" {
		p, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid page")
			return
		}
		page = p
	}

	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}

	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()

	words, err := h.dict.GetWords(ctx, user.ID, limit, page, order)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get words")
		slogger.Log.ErrorContext(ctx, "Failed to get words", "err", err)
		return
	}
	JSONResponse(w, http.StatusOK, words)
}

func (h *Handler) AddWords(w http.ResponseWriter, r *http.Request) {
	var req models.DictionaryWord

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()
	slogger.Log.DebugContext(ctx, "Adding words", "req", req)

	err := h.dict.AddWords(ctx, req)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to add words")
		slogger.Log.ErrorContext(ctx, "Failed to add words", "err", err)
		return
	}
	JSONResponse(w, http.StatusOK, nil) //todo
}

func (h *Handler) SearchDictionary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		JSONResponse(w, http.StatusOK, []models.DictionaryWord{})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()

	words, err := h.dict.SearchWords(ctx, query)
	if err != nil {
		slogger.Log.ErrorContext(ctx, "Search failed", "err", err)
		WriteError(w, http.StatusInternalServerError, "Search failed")
		return
	}

	JSONResponse(w, http.StatusOK, words)
}

type AddToLearningRequest struct {
	WordID string `json:"word_id"`
}

func (h *Handler) AddWordToLearning(w http.ResponseWriter, r *http.Request) {
	var req AddToLearningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	// Получаем ID текущего юзера из контекста (AuthMiddleware)
	user := r.Context().Value(middleware.UserCtxKey{}).(*models.User)

	ctx, cancel := context.WithTimeout(r.Context(), ctxWithTimeout)
	defer cancel()

	err := h.dict.AddWordToLearning(ctx, user.ID, req.WordID)
	if err != nil {
		slogger.Log.ErrorContext(ctx, "Failed to add word to learning", "err", err)
		// Упрощенно 500, но лучше проверять на дубликаты
		WriteError(w, http.StatusInternalServerError, "Failed to add word")
		return
	}

	JSONResponse(w, http.StatusCreated, map[string]string{"status": "added"})
}
