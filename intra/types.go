package intra

import "time"

type Campus struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	TimeZone string `json:"time_zone"`
	Language struct {
		ID         int       `json:"id"`
		Name       string    `json:"name"`
		Identifier string    `json:"identifier"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
	} `json:"language"`
	UsersCount  int `json:"users_count"`
	VogsphereID int `json:"vogsphere_id"`
}

type Cursus struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
}

type Skill struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectShort struct {
	ID   int `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Project struct {
	ProjectShort
	Description string        `json:"description"`
	Parent      *ProjectShort   `json:"parent"`
	Children    []ProjectShort `json:"children"`
	Objectives  []string      `json:"objectives"`
	Tier        int           `json:"tier"`
	Attachments []interface{} `json:"attachments"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Exam        bool          `json:"exam"`
	Cursus      []Cursus      `json:"cursus"`
	Campus      []Campus      `json:"campus"`
	Skills      []Skill       `json:"skills"`
	Videos      []interface{} `json:"videos"`
	Tags []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"tags"`
	ProjectSessions []struct {
		ID               int         `json:"id"`
		Solo             bool        `json:"solo"`
		BeginAt          interface{} `json:"begin_at"`
		EndAt            interface{} `json:"end_at"`
		EstimateTime     int         `json:"estimate_time"`
		DurationDays     interface{} `json:"duration_days"`
		TerminatingAfter interface{} `json:"terminating_after"`
		ProjectID        int         `json:"project_id"`
		CampusID         *int        `json:"campus_id"`
		CursusID         *int        `json:"cursus_id"`
		CreatedAt        time.Time   `json:"created_at"`
		UpdatedAt        time.Time   `json:"updated_at"`
		MaxPeople        interface{} `json:"max_people"`
		IsSubscriptable  bool        `json:"is_subscriptable"`
		Scales []struct {
			ID               int  `json:"id"`
			CorrectionNumber int  `json:"correction_number"`
			IsPrimary        bool `json:"is_primary"`
		} `json:"scales"`
		Uploads []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"uploads"`
		TeamBehaviour string `json:"team_behaviour"`
	} `json:"project_sessions"`
}

type ProjectUser struct {
	ID            int `json:"id" lua:"id"`
	Occurrence    int `json:"occurrence" lua:"occurrence"`
	FinalMark     int `json:"final_mark"`
	Status        string `json:"status"`
	Validated     bool `json:"validated?"`
	CurrentTeamID int `json:"current_team_id"`
	Project       ProjectShort `json:"project"`
	CursusIds     []int `json:"cursus_ids"`
	User          UserShort `json:"user"`
	Teams []struct {
		ID            int `json:"id"`
		Name          string `json:"name"`
		URL           string `json:"url"`
		FinalMark     int `json:"final_mark"`
		ProjectID     int `json:"project_id"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
		Status        string `json:"status"`
		TerminatingAt time.Time `json:"terminating_at"`
		Users []struct {
			UserShort
			Leader         bool `json:"leader"`
			Occurrence     int `json:"occurrence"`
			Validated      bool `json:"validated"`
			ProjectsUserID int `json:"projects_user_id"`
		} `json:"users"`
		Locked           bool `json:"locked?"`
		Validated        bool `json:"validated?"`
		Closed           bool `json:"closed?"`
		RepoURL          string `json:"repo_url"`
		RepoUUID         string `json:"repo_uuid"`
		LockedAt         time.Time `json:"locked_at"`
		ClosedAt         time.Time `json:"closed_at"`
		ProjectSessionID int `json:"project_session_id"`
	} `json:"teams"`
}

type UserShort struct {
	ID    int `json:"id"`
	Login string `json:"login"`
	URL   string `json:"url"`
}
