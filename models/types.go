package models

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AssignedTo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Issue struct {
	ID         string      `json:"id"`
	ShortID    string      `json:"short_id"`
	Title      string      `json:"title"`
	Status     string      `json:"status"`
	Level      string      `json:"level"`
	Project    Project     `json:"project"`
	Count      string      `json:"count"`
	UserCount  int         `json:"user_count"`
	FirstSeen  string      `json:"first_seen"`
	LastSeen   string      `json:"last_seen"`
	Reporter   string      `json:"reporter"`
	AssignedTo *AssignedTo `json:"assigned_to"`
	Source     string      `json:"source"`
	URL        string      `json:"url"`
}
