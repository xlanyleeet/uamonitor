package main

import "time"

type Status struct {
	Date        time.Time `json:"date"`
	IsOnline    bool      `json:"isOnline"`
	OnlineCount *int      `json:"onlineCount"`
	MaxOnline   *int      `json:"maxOnline"`
	Version     *string   `json:"version"`
}

type Server struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	IP          string   `json:"ip"`
	Port        string   `json:"port"`
	Icon        *string  `json:"icon"`
	ChangeIcon  bool     `json:"changeIcon"`
	Telegram    *string  `json:"telegram"`
	Discord     *string  `json:"discord"`
	Youtube     *string  `json:"youtube"`
	Link        *string  `json:"link"`
	Description *string  `json:"description"`
	Statuses    []Status `json:"statuses"`
	Uptime      Uptime   `json:"uptime"`
}

type Point struct {
	IsOnline    bool      `json:"isOnline"`
	OnlineCount int       `json:"onlineCount"`
	Date        time.Time `json:"date"`
}

type Graph struct {
	ID   int     `json:"id"`
	Data []Point `json:"data"`
}

// Uptime is estimated availability over observed time, excluding monitoring gaps.
type Uptime struct {
	Days     int      `json:"days"`
	Percent  *float64 `json:"percent"`
	Coverage float64  `json:"coverage"`
}
