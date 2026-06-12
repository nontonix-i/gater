package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/user/gater/internal/auth"
	"github.com/user/gater/internal/model"
	"github.com/user/gater/internal/provider"
	"github.com/user/gater/internal/task"
)

type Handler struct {
	taskManager *task.Manager
	registry    *provider.Registry
	db          *gorm.DB
	authService *auth.Service
}

func NewHandler(tm *task.Manager, reg *provider.Registry, db *gorm.DB) *Handler {
	return &Handler{
		taskManager: tm,
		registry:    reg,
		db:          db,
		authService: auth.NewService(db),
	}
}

func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	providers := r.Form["providers"]
	if len(providers) == 0 {
		writeError(w, http.StatusBadRequest, "at least one provider is required")
		return
	}

	title := r.FormValue("title")
	if title == "" {
		title = header.Filename
	}

	req := &task.CreateRequest{
		UserID:     userID,
		SourceType: "direct_upload",
		FileName:   header.Filename,
		FileSize:   header.Size,
		Title:      title,
		File:       file,
		Providers:  providers,
	}

	t, err := h.taskManager.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"task_id": t.ID,
		"status":  t.Status,
	})
}

func (h *Handler) UploadURL(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	var req struct {
		URL       string   `json:"url"`
		Providers []string `json:"providers"`
		Title     string   `json:"title"`
		FileName  string   `json:"file_name"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	if len(req.Providers) == 0 {
		writeError(w, http.StatusBadRequest, "at least one provider is required")
		return
	}

	filename := req.FileName
	if filename == "" {
		if req.Title != "" {
			filename = req.Title
		} else {
			filename = "remote_file"
		}
	}

	taskReq := &task.CreateRequest{
		UserID:     userID,
		SourceType: "remote_url",
		SourceURL:  req.URL,
		FileName:   filename,
		Title:      req.Title,
		Providers:  req.Providers,
	}

	t, err := h.taskManager.Create(r.Context(), taskReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"task_id": t.ID,
		"status":  t.Status,
	})
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	t, err := h.taskManager.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	resp := taskResponse{
		ID:          t.ID,
		Status:      t.Status,
		Title:       t.Title,
		FileName:    t.FileName,
		FileSize:    t.FileSize,
		SourceType:  t.SourceType,
		SourceURL:   t.SourceURL,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
		Results:     make([]resultResponse, 0),
		Metadata:    make(map[string]string),
	}

	for _, r := range t.Results {
		resp.Results = append(resp.Results, resultResponse{
			Provider:         r.Provider,
			Status:           r.Status,
			Progress:         r.Progress,
			SourceURL:        r.SourceURL,
			OutputURL:        r.OutputURL,
			FileCode:         r.FileCode,
			ProviderFileName: r.ProviderFileName,
			ProviderFileSize: r.ProviderFileSize,
			Error:            r.ErrorMessage,
			StartedAt:        r.StartedAt,
			CompletedAt:      r.CompletedAt,
		})
	}

	for _, m := range t.Metadata {
		resp.Metadata[m.Key] = m.Value
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) StreamProgress(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	ctx := r.Context()
	done := false

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var task model.Task
		if err := h.db.Preload("Results").First(&task, "id = ?", taskID).Error; err != nil {
			fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%s\"}\n\n", err.Error())
			flusher.Flush()
			return
		}

		var results []resultResponse
		allDone := true
		for _, r := range task.Results {
			res := resultResponse{
				Provider:         r.Provider,
				Status:           r.Status,
				Progress:         r.Progress,
				SourceURL:        r.SourceURL,
				OutputURL:        r.OutputURL,
				FileCode:         r.FileCode,
				ProviderFileName: r.ProviderFileName,
				ProviderFileSize: r.ProviderFileSize,
				Error:            r.ErrorMessage,
				StartedAt:        r.StartedAt,
				CompletedAt:      r.CompletedAt,
			}
			results = append(results, res)
			if r.Status != "completed" && r.Status != "failed" {
				allDone = false
			}
		}

		resp := map[string]interface{}{
			"id":       task.ID,
			"status":   task.Status,
			"results":  results,
		}

		data, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if allDone {
			if !done {
				done = true
				continue
			}
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	}
}

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	tasks, total, err := h.taskManager.ListTasks(userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	list := make([]taskResponse, 0)
	for _, t := range tasks {
		resp := taskResponse{
			ID:          t.ID,
			Status:      t.Status,
			Title:       t.Title,
			FileName:    t.FileName,
			FileSize:    t.FileSize,
			SourceType:  t.SourceType,
			SourceURL:   t.SourceURL,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
			CompletedAt: t.CompletedAt,
			Results:     make([]resultResponse, 0),
			Metadata:    make(map[string]string),
		}
		for _, r := range t.Results {
			resp.Results = append(resp.Results, resultResponse{
				Provider:         r.Provider,
				Status:           r.Status,
				Progress:         r.Progress,
				SourceURL:        r.SourceURL,
				OutputURL:        r.OutputURL,
				FileCode:         r.FileCode,
				ProviderFileName: r.ProviderFileName,
				ProviderFileSize: r.ProviderFileSize,
				Error:            r.ErrorMessage,
				StartedAt:        r.StartedAt,
				CompletedAt:      r.CompletedAt,
			})
		}
		list = append(list, resp)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tasks":  list,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	list := h.registry.List()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"providers": list,
	})
}

// --- Auth endpoints ---

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password required")
		return
	}

	resp, err := h.authService.Register(r.Context(), auth.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err == auth.ErrEmailTaken {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password required")
		return
	}

	resp, err := h.authService.Login(r.Context(), auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err == auth.ErrInvalidCreds {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.authService.GetUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// --- Settings ---

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())

	var user model.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	defaultProviders := []string{}
	if user.DefaultProviders != nil {
		json.Unmarshal(user.DefaultProviders, &defaultProviders)
	}

	resp := map[string]interface{}{
		"default_providers": defaultProviders,
		"api_key":           user.APIKey,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())

	var req struct {
		DefaultProviders []string `json:"default_providers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	data, _ := json.Marshal(req.DefaultProviders)
	h.db.Model(&model.User{}).Where("id = ?", userID).Update("default_providers", data)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Provider Credentials ---

type credentialField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

var providerCredentialFields = map[string][]credentialField{
	"abyss":        {{"api_key", "API Key"}},
	"doodstream":   {{"api_key", "API Key"}, {"folder_id", "Folder ID"}},
	"gofile":       {{"token", "Token"}, {"folder_id", "Folder ID"}},
	"lulustream":   {{"api_key", "API Key"}, {"folder_id", "Folder ID"}, {"category_id", "Category ID"}},
	"rapidgator":   {{"username", "Username"}, {"password", "Password"}, {"folder_id", "Folder ID"}},
	"rpmshare":     {{"api_token", "API Token"}, {"folder_id", "Folder ID"}},
	"seekstreaming": {{"api_token", "API Token"}, {"folder_id", "Folder ID"}},
	"streamtape":   {{"login", "Login"}, {"key", "Key"}},
	"turboviplay":  {{"api_key", "API Key"}, {"folder_id", "Folder ID"}},
	"vidoza":       {{"api_token", "API Token"}, {"category_id", "Category ID"}, {"folder_id", "Folder ID"}},
	"vikingfiles":  {{"user", "User ID"}},
}

func (h *Handler) GetProviderCredentials(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())
	providerName := chi.URLParam(r, "name")

	fields, ok := providerCredentialFields[providerName]
	if !ok {
		fields = []credentialField{}
	}

	var cred model.ProviderCredential
	hasCred := h.db.Where("user_id = ? AND provider = ?", userID, providerName).First(&cred).Error == nil

	type fieldValue struct {
		Key      string `json:"key"`
		Label    string `json:"label"`
		HasValue bool   `json:"has_value"`
	}

	result := make([]fieldValue, 0, len(fields))
	for _, f := range fields {
		hasVal := false
		if hasCred {
			var parsed map[string]string
			if err := json.Unmarshal(cred.Credentials, &parsed); err == nil {
				_, hasVal = parsed[f.Key]
			}
		}
		result = append(result, fieldValue{
			Key:      f.Key,
			Label:    f.Label,
			HasValue: hasVal,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"provider":   providerName,
		"has_creds":  hasCred,
		"fields":     result,
	})
}

func (h *Handler) UpdateProviderCredentials(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())
	providerName := chi.URLParam(r, "name")

	var req struct {
		Values map[string]string `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	data, _ := json.Marshal(req.Values)

	var existing model.ProviderCredential
	err := h.db.Where("user_id = ? AND provider = ?", userID, providerName).First(&existing).Error
	if err == nil {
		h.db.Model(&existing).Update("credentials", data)
	} else {
		cred := &model.ProviderCredential{
			ID:          uuid.New().String(),
			UserID:      userID,
			Provider:    providerName,
			Credentials: data,
		}
		h.db.Create(cred)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Regenerate API Key ---

func (h *Handler) RegenerateKey(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())

	key := fmt.Sprintf("gater-key-%s", uuid.New().String()[:8])
	h.db.Model(&model.User{}).Where("id = ?", userID).Update("api_key", key)

	writeJSON(w, http.StatusOK, map[string]string{"api_key": key})
}

// --- Individual provider detail ---

func (h *Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	prov, err := h.registry.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}

	writeJSON(w, http.StatusOK, provider.ProviderInfo{
		Name:   prov.Name(),
		Type:   prov.Type(),
		Anon:   prov.SupportsAnonymous(),
		Remote: prov.SupportsRemoteURL(),
		HasAPI: prov.HasAPI(),
	})
}
