package health

import (
	"context"
	"fmt"
	"time"
)

var (
	// ensure Probe implements Prober
	_ = Prober(&Probe{})
)

type Prober interface {
	// ServiceID unique ID for the dependency
	ServiceID() string
	// AffectsStatus key of the status which will be affected if the dependency fails
	AffectsStatuses() []Statuskey
	Checker
}

type Checker interface {
	// Check returns the status of the probed service
	Check(ctx context.Context) error
}

type CheckerFunc func(ctx context.Context) error

func (cf CheckerFunc) Check(ctx context.Context) error {
	return cf(ctx)
}

type Probe struct {
	ID               string
	AffectedStatuses []Statuskey
	Checker          Checker
}

func (pr *Probe) ServiceID() string {
	return pr.ID
}

func (pr *Probe) AffectsStatuses() []Statuskey {
	return pr.AffectedStatuses
}

func (pr *Probe) Check(ctx context.Context) error {
	if pr.Checker == nil {
		return nil
	}

	return pr.Checker.Check(ctx)
}

type DependencyStatus struct {
	ServiceID        string
	Status           string
	AffectedStatuses []Statuskey
	AsOf             time.Time
}

func ProbeDependencies(
	timeout time.Duration,
	probers ...Prober,
) []DependencyStatus {
	total := len(probers)
	statuses := make(chan DependencyStatus, total)
	healthOK := HealthOK.String()
	healthNotOK := HealthNotOK.String()

	for i := range probers {
		go func(pinger Prober) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			hc := DependencyStatus{
				ServiceID:        pinger.ServiceID(),
				Status:           healthOK,
				AffectedStatuses: pinger.AffectsStatuses(),
			}
			err := pinger.Check(ctx)
			hc.AsOf = time.Now()
			if err != nil {
				hc.Status = healthNotOK
			}
			statuses <- hc
		}(probers[i])
	}

	list := make([]DependencyStatus, 0, total)
	for h := range statuses {
		list = append(list, h)
		if len(list) >= total {
			break
		}
	}
	close(statuses)

	return list
}

type Stopper interface {
	Stop()
}

func Start(
	delay time.Duration,
	pstatus *ProbeResponder,
	pingers ...Prober,
) Stopper {
	if len(pingers) == 0 {
		return nil
	}

	/*
		Important: having regular pings would keep the respective clients "active".
		This may or may not be a desirable behavior. For e.g. it might be better
		to let all connections of MongoDB be disconnected if there's no activity, so that
		the server would only need to deal with fewer connections
	*/
	tick := time.NewTicker(delay)
	go func() {
		// execute the probe immediately so that we don't have to wait for the first tick
		probe(delay, pstatus, pingers...)
		for range tick.C {
			probe(delay, pstatus, pingers...)
		}
	}()
	return tick
}

func probe(delay time.Duration, pstatus *ProbeResponder, pingers ...Prober) {
	startupOK := true
	readyOK := true
	liveOK := true

	for _, hc := range ProbeDependencies(delay, pingers...) {
		pstatus.AppendHealthResponse(
			hc.ServiceID,
			fmt.Sprintf("%s: %s", hc.Status, hc.AsOf.Format(time.RFC3339)),
		)

		ok := IsHealthOK(hc.Status)
		for _, afStatus := range hc.AffectedStatuses {
			switch afStatus {
			case StatusStartup:
				startupOK = startupOK && ok
			case StatusReady:
				readyOK = readyOK && ok
			case StatusLive:
				liveOK = liveOK && ok
			}
		}
	}

	pstatus.SetNotStarted(!startupOK)
	pstatus.SetNotReady(!readyOK)
	pstatus.SetNotLive(!liveOK)
}
