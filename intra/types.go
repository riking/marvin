package intra

import "time"

type Campus struct {
	ID       int    `json:"id" lua:"id"`
	Name     string `json:"name" lua:"name"`
	TimeZone string `json:"time_zone" lua:"time_zone"`
	Language struct {
		ID         int       `json:"id" lua:"id"`
		Name       string    `json:"name" lua:"name"`
		Identifier string    `json:"identifier" lua:"identifier"`
		CreatedAt  time.Time `json:"created_at" lua:"created_at"`
		UpdatedAt  time.Time `json:"updated_at" lua:"updated_at"`
	} `json:"language" lua:"language"`
	UsersCount  int `json:"users_count" lua:"users_count"`
	VogsphereID int `json:"vogsphere_id" lua:"vogsphere_id"`
}

type Cursus struct {
	ID        int       `json:"id" lua:"id"`
	CreatedAt time.Time `json:"created_at" lua:"created_at"`
	Name      string    `json:"name" lua:"name"`
	Slug      string    `json:"slug" lua:"slug"`
}

type Skill struct {
	ID        int       `json:"id" lua:"id"`
	Name      string    `json:"name" lua:"name"`
	CreatedAt time.Time `json:"created_at" lua:"created_at"`
}

type ProjectShort struct {
	ID   int    `json:"id" lua:"id"`
	Name string `json:"name" lua:"name"`
	Slug string `json:"slug" lua:"slug"`
}

type Project struct {
	ProjectShort
	Description string         `json:"description"`
	Parent      *ProjectShort  `json:"parent"`
	Children    []ProjectShort `json:"children"`
	Objectives  []string       `json:"objectives"`
	Tier        int            `json:"tier"`
	Attachments []interface{}  `json:"attachments"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Exam        bool           `json:"exam"`
	Cursus      []Cursus       `json:"cursus"`
	Campus      []Campus       `json:"campus"`
	Skills      []Skill        `json:"skills"`
	Videos      []interface{}  `json:"videos"`
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
	ID            int          `json:"id" lua:"id"`
	Occurrence    int          `json:"occurrence" lua:"occurrence"`
	FinalMark     int          `json:"final_mark" lua:"final_mark"`
	Status        string       `json:"status" lua:"status"`
	Validated     bool         `json:"validated?" lua:"validated"`
	CurrentTeamID int          `json:"current_team_id" lua:"current_team_id"`
	Project       ProjectShort `json:"project" lua:"project"`
	CursusIds     []int        `json:"cursus_ids" lua:"cursus_ids"`
	User          UserShort    `json:"user" lua:"user"`
	Teams []struct {
		ID            int       `json:"id" lua:"id"`
		Name          string    `json:"name" lua:"name"`
		URL           string    `json:"url" lua:"url"`
		FinalMark     int       `json:"final_mark" lua:"final_mark"`
		ProjectID     int       `json:"project_id" lua:"project_id"`
		CreatedAt     time.Time `json:"created_at" lua:"created_at"`
		UpdatedAt     time.Time `json:"updated_at" lua:"updated_at"`
		Status        string    `json:"status" lua:"status"`
		TerminatingAt *time.Time `json:"terminating_at" lua:"terminating_at"`
		Users []struct {
			UserShort
			Leader         bool `json:"leader" lua:"leader"`
			Occurrence     int  `json:"occurrence" lua:"occurrence"`
			Validated      bool `json:"validated" lua:"validated"`
			ProjectsUserID int  `json:"projects_user_id" lua:"projects_user_id"`
		} `json:"users" lua:"users"`
		Locked           bool      `json:"locked?" lua:"locked"`
		Validated        bool      `json:"validated?" lua:"validated"`
		Closed           bool      `json:"closed?" lua:"closed"`
		RepoURL          string    `json:"repo_url" lua:"repo_url"`
		RepoUUID         string    `json:"repo_uuid" lua:"repo_uuid"`
		LockedAt         *time.Time `json:"locked_at" lua:"locked_at"`
		ClosedAt         *time.Time `json:"closed_at" lua:"closed_at"`
		ProjectSessionID int       `json:"project_session_id" lua:"project_session_id"`
	} `json:"teams" lua:"teams"`
}

type UserShort struct {
	ID    int    `json:"id" lua:"id"`
	Login string `json:"login" lua:"login"`
	URL   string `json:"url" lua:"url"`
}

type User struct {
	Achievements []interface{} `json:"achievements"`
	Campus []struct {
		ID int `json:"id"`
		Language struct {
			CreatedAt time.Time `json:"created_at"`
			ID int `json:"id"`
			Identifier string `json:"identifier"`
			Name string `json:"name"`
			UpdatedAt time.Time `json:"updated_at"`
		} `json:"language"`
		Name string `json:"name"`
		TimeZone string `json:"time_zone"`
		UsersCount int `json:"users_count"`
		VogsphereID int `json:"vogsphere_id"`
	} `json:"campus"`
	CampusUsers []struct {
		CampusID int `json:"campus_id"`
		ID int `json:"id"`
		IsPrimary bool `json:"is_primary"`
		UserID int `json:"user_id"`
	} `json:"campus_users"`
	CorrectionPoint int `json:"correction_point"`
	CursusUsers []struct {
		BeginAt time.Time `json:"begin_at"`
		Cursus struct {
			CreatedAt time.Time `json:"created_at"`
			ID int `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"cursus"`
		CursusID int `json:"cursus_id"`
		EndAt *time.Time `json:"end_at"`
		Grade string `json:"grade"`
		ID int `json:"id"`
		Level float64 `json:"level"`
		Skills []struct {
			ID int `json:"id"`
			Level float64 `json:"level"`
			Name string `json:"name"`
		} `json:"skills"`
		User struct {
			ID int `json:"id"`
			Login string `json:"login"`
			URL string `json:"url"`
		} `json:"user"`
	} `json:"cursus_users"`
	Displayname string `json:"displayname"`
	Email string `json:"email"`
	ExpertisesUsers []struct {
		ContactMe bool `json:"contact_me"`
		CreatedAt time.Time `json:"created_at"`
		ExpertiseID int `json:"expertise_id"`
		ID int `json:"id"`
		Interested bool `json:"interested"`
		UserID int `json:"user_id"`
		Value int `json:"value"`
	} `json:"expertises_users"`
	FirstName string `json:"first_name"`
	Groups []interface{} `json:"groups"`
	ID int `json:"id"`
	ImageURL string `json:"image_url"`
	LastName string `json:"last_name"`
	Location interface{} `json:"location"`
	Login string `json:"login"`
	Partnerships []interface{} `json:"partnerships"`
	Patroned []interface{} `json:"patroned"`
	Patroning []interface{} `json:"patroning"`
	Phone string `json:"phone"`
	PoolMonth string `json:"pool_month"`
	PoolYear string `json:"pool_year"`
	ProjectsUsers []struct {
		CurrentTeamID int `json:"current_team_id"`
		CursusIds []int `json:"cursus_ids"`
		FinalMark int `json:"final_mark"`
		ID int `json:"id"`
		Occurrence int `json:"occurrence"`
		Project struct {
			ID int `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"project"`
		Status string `json:"status"`
		Validated bool `json:"validated?"`
	} `json:"projects_users"`
	Staff bool `json:"staff?"`
	Titles []interface{} `json:"titles"`
	URL string `json:"url"`
	Wallet int `json:"wallet"`
}