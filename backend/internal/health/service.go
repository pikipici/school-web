package health

import "context"

type Pinger interface {
	Ping(context.Context) error
}

type Service struct {
	pinger Pinger
}

type Readiness struct {
	Ready    bool   `json:"-"`
	Status   string `json:"status"`
	Database string `json:"database"`
	Error    string `json:"-"`
}

func NewService(pinger Pinger) Service {
	return Service{pinger: pinger}
}

func (s Service) Readiness(ctx context.Context) Readiness {
	if s.pinger == nil || s.pinger.Ping(ctx) != nil {
		return Readiness{Status: "unavailable", Database: "unavailable"}
	}
	return Readiness{Ready: true, Status: "ok", Database: "ok"}
}
