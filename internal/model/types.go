package model

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	ID              string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email           string `gorm:"uniqueIndex"`
	PasswordHash    string
	Name            string
	APIKey          string `gorm:"uniqueIndex;not null"`
	DefaultProviders datatypes.JSON
	CreatedAt       time.Time `gorm:"autoCreateTime"`
}

func (User) TableName() string { return "users" }

type ProviderCredential struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      string         `gorm:"type:uuid;not null;index"`
	Provider    string         `gorm:"not null"`
	Credentials datatypes.JSON `gorm:"not null"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`

	User User `gorm:"constraint:OnDelete:CASCADE"`
}

func (ProviderCredential) TableName() string { return "provider_credentials" }

type Task struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      string     `gorm:"type:uuid;index"`
	Status      string     `gorm:"default:pending"`
	SourceType  string     `gorm:"not null"`
	SourceURL   string
	Title       string
	FileName    string `gorm:"not null"`
	FileSize    int64  `gorm:"not null;default:0"`
	FilePath    string
	Providers   datatypes.JSON
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	CompletedAt *time.Time

	User     User           `gorm:"constraint:OnDelete:CASCADE"`
	Results  []TaskResult   `gorm:"foreignKey:TaskID"`
	Metadata []TaskMetadata `gorm:"foreignKey:TaskID"`
}

func (Task) TableName() string { return "tasks" }

type TaskResult struct {
	ID               string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TaskID           string     `gorm:"type:uuid;not null;index"`
	Provider         string     `gorm:"not null"`
	Status           string     `gorm:"default:pending"`
	SourceURL        string
	OutputURL        string
	FileCode         string
	ProviderFileName string
	ProviderFileSize int64
	Progress         int        `gorm:"default:0"`
	ErrorMessage     string
	StartedAt        *time.Time
	CompletedAt      *time.Time
	LastKeepaliveAt  *time.Time
	KeepaliveCount   int        `gorm:"default:0"`

	Task Task `gorm:"constraint:OnDelete:CASCADE"`
}

func (TaskResult) TableName() string { return "task_results" }

type TaskMetadata struct {
	ID     string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TaskID string `gorm:"type:uuid;not null;uniqueIndex:idx_task_meta_key"`
	Key    string `gorm:"not null;uniqueIndex:idx_task_meta_key"`
	Value  string `gorm:"not null"`

	Task Task `gorm:"constraint:OnDelete:CASCADE"`
}

func (TaskMetadata) TableName() string { return "task_metadata" }
