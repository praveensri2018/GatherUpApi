package main

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey"`
	UID       string `gorm:"type:char(36);uniqueIndex;not null"`
	Email     string `gorm:"uniqueIndex;not null"`
	Password  string `gorm:"not null"`
	Name      string
	CreatedAt time.Time
}

type Event struct {
	ID              uint   `gorm:"primaryKey"`
	UUID            string `gorm:"type:char(36);uniqueIndex;not null"`
	HostUID         string `gorm:"type:char(36);index;not null"`
	GameName        string
	DateIso         time.Time
	Location        string
	IsOnline        bool
	Description     string
	MaxParticipants int
	Status          string
	Results         string `gorm:"type:json"`
	CreatedAt       time.Time
}

type JoinRequest struct {
	ID        uint   `gorm:"primaryKey"`
	UUID      string `gorm:"type:char(36);uniqueIndex;not null"`
	EventUUID string `gorm:"type:char(36);index;not null"`
	UserUID   string `gorm:"type:char(36);index;not null"`
	Message   string
	Status    string
	CreatedAt time.Time
}
