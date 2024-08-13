package config

import "time"

type Version struct {
	Checksum string    `json:"checksum"`
	DateTime time.Time `json:"dateTime"`
}

type Config struct {
	Port    int    `json:"port"`
	Host    string `json:"host"`
	LogPath string `json:"logPath"`
	Redis   Redis  `json:"redis"`
}

type Redis struct {
	Port     int    `json:"port"`
	Host     string `json:"host"`
	Password string `json:"-"`
}
