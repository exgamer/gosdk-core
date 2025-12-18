package config

import "github.com/google/uuid"

// AppInfo Данные приложения
type AppInfo struct {
	RequestId        string
	AppsflyerId      string
	AuthToken        string
	CompanyIds       []int
	UserId           int
	CompanyId        int
	CurrentCompanyId int
	CityId           int
	LanguageCode     string
	Iin              string
	RequestScheme    string
	RequestHost      string
	RequestMethod    string
	RequestUrl       string
	ServiceName      string
	AppEnv           string
	CacheControl     string
}

func (s *AppInfo) GenerateRequestId() {
	s.RequestId = uuid.New().String()
}

func (s *AppInfo) SetConsoleMode(name string) {
	s.RequestId = uuid.New().String()
	s.RequestMethod = "console"
	s.RequestUrl = name
	s.GenerateRequestId()
}
