package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/user/gater/internal/auth"
	"github.com/user/gater/internal/model"
)

func (h *Handler) SaveCredential(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())

	var req struct {
		Provider    string            `json:"provider"`
		Credentials map[string]string `json:"credentials"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	data, _ := json.Marshal(req.Credentials)

	var existing model.ProviderCredential
	err := h.db.Where("user_id = ? AND provider = ?", userID, req.Provider).First(&existing).Error

	if err == nil {
		h.db.Model(&existing).Update("credentials", data)
	} else {
		cred := &model.ProviderCredential{
			ID:          uuid.New().String(),
			UserID:      userID,
			Provider:    req.Provider,
			Credentials: data,
		}
		h.db.Create(cred)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (h *Handler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserID(r.Context())

	var creds []model.ProviderCredential
	if err := h.db.Where("user_id = ?", userID).Find(&creds).Error; err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var list []map[string]interface{}
	for _, c := range creds {
		list = append(list, map[string]interface{}{
			"provider":   c.Provider,
			"updated_at": c.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"credentials": list})
}
