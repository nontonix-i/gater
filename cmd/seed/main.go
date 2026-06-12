package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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

	apiKey := "gater-key-001"
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

	creds := []struct {
		provider string
		data     map[string]string
	}{
		{"gofile", map[string]string{"token": "SWXawrVlfs3ELuzASM2zVCb6X26OSaSk"}},
		{"streamtape", map[string]string{"login": "31fb29c839b2821b199f", "key": "O6DPvpxWP1CZ36e"}},
		{"turboviplay", map[string]string{"api_key": "bnhrC5bsr8"}},
		{"rapidgator", map[string]string{"username": "fiqriky@gmail.com", "password": "ciamis17"}},
		{"rpmshare", map[string]string{"api_token": "a50fcaba0b3101798b3396db"}},
		{"vikingfiles", map[string]string{"user": "6x4WbuoM5b"}},
		{"doodstream", map[string]string{"api_key": "533356fpbu99umaax8vw5m"}},
		{"abyss", map[string]string{"api_key": "c10afbc1d7e668cc43a77250d89df61f"}},
		{"lulustream", map[string]string{"api_key": "290956gu7mbuaqmkq1to25"}},
		{"seekstreaming", map[string]string{"api_token": "4731ad2d882f3500d73134b3"}},
		{"vidoza", map[string]string{"api_token": "rydpztpiovbdgt7571kyezq5kxpti6p8agp0zokd59ark9ruio36q2fcoqhg"}},
	}

	for _, c := range creds {
		data, _ := json.Marshal(c.data)
		var existing model.ProviderCredential
		err := db.Where("user_id = ? AND provider = ?", user.ID, c.provider).First(&existing).Error
		if err == nil {
			db.Model(&existing).Update("credentials", data)
			fmt.Printf("Updated: %s\n", c.provider)
		} else {
			cred := &model.ProviderCredential{
				ID:          uuid.New().String(),
				UserID:      user.ID,
				Provider:    c.provider,
				Credentials: data,
			}
			db.Create(cred)
			fmt.Printf("Created: %s\n", c.provider)
		}
	}

	fmt.Println("\nDone! API Key:", apiKey)
	os.Exit(0)
}
