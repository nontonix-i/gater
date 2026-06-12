package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/user/gater/config"
	"github.com/user/gater/internal/model"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	apiKey := os.Getenv("SEED_API_KEY")
	if apiKey == "" {
		apiKey = "gater-key-001"
	}

	var user model.User
	err = db.Where("api_key = ?", apiKey).First(&user).Error
	if err != nil {
		user = model.User{
			ID:     uuid.New().String(),
			APIKey: apiKey,
			Name:   "main",
		}
		db.Create(&user)
		fmt.Println("User created:", user.ID)
	} else {
		fmt.Println("User exists:", user.ID)
	}

	providers := []string{
		"abyss", "anonmp4", "doodstream", "gofile", "lulustream",
		"rapidgator", "rpmshare", "seekstreaming", "streamtape",
		"turboviplay", "vidoza", "vikingfiles",
	}

	for _, p := range providers {
		envKey := "SEED_" + strings.ToUpper(p)
		raw := os.Getenv(envKey)
		if raw == "" {
			continue
		}

		data := make(map[string]string)
		for _, pair := range strings.Split(raw, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				data[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		if len(data) == 0 {
			continue
		}

		credJSON, _ := json.Marshal(data)
		var existing model.ProviderCredential
		err := db.Where("user_id = ? AND provider = ?", user.ID, p).First(&existing).Error
		if err == nil {
			db.Model(&existing).Update("credentials", credJSON)
			fmt.Printf("Updated: %s\n", p)
		} else {
			cred := &model.ProviderCredential{
				ID:          uuid.New().String(),
				UserID:      user.ID,
				Provider:    p,
				Credentials: credJSON,
			}
			db.Create(cred)
			fmt.Printf("Created: %s\n", p)
		}
	}

	fmt.Println("\nDone! API Key:", apiKey)
	os.Exit(0)
}
