package src

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type LB_STATE int

// Loadbalancer states
const (
	LB_STATE_INIT    LB_STATE = 0
	LB_STATE_ACTIVE  LB_STATE = 1
	LB_STATE_CLOSING LB_STATE = 2
	LB_STATE_CLOSED  LB_STATE = 3
)

type CustomHeader struct {
	Name  string
	Value string
}
type CustomHeaderRule struct {
	Method  string
	Headers []CustomHeader
}

type BalancerDebugger struct {
	DebugMode                   bool
	DeubgDataMutext             *sync.Mutex
	DebugIndicesHistory         []int
	DebugWeightedIndicesHistory [][2]int
}

func (bd *BalancerDebugger) recordIndex(index int) {
	bd.DeubgDataMutext.Lock()
	bd.DebugIndicesHistory = append(bd.DebugIndicesHistory, index)
	bd.DeubgDataMutext.Unlock()
}
func (bd *BalancerDebugger) recordWeightedIndex(index int, weight int) {
	bd.DeubgDataMutext.Lock()
	bd.DebugWeightedIndicesHistory = append(bd.DebugWeightedIndicesHistory, [2]int{index, weight})
	bd.DeubgDataMutext.Unlock()
}

type Balancer struct {
	liveConnections   sync.WaitGroup
	Id                string
	Mode              string
	RoutePrefix       string
	TargetWaitTimeout time.Duration
	Targets           []*Target
	State             LB_STATE
	CustomHeaderRules []CustomHeaderRule
	// NextAvailableServer func(lb *Balancer) *Target
	Logic BalancerLogic
	BalancerDebugger
}

var LoadBalancersPool map[string]*Balancer

func (lb *Balancer) SetBalancerLogic() {

	switch lb.Mode {
	case LB_MODE_ROUNDROBIN:
		lb.Logic = &RoundRobinLogic{}
	case LB_MODE_WEIGHTED_ROUNDROBIN:
		lb.Logic = &WeightedRoundRobinLogic{}
	case LB_MODE_RANDOM:
		lb.Logic = &RandomLogic{}
	case LB_MODE_LEAST_CONNECTIONS_RANDOM:
		lb.Logic = &LeastConnectionsRandomLogic{}
	case LB_MODE_LEAST_CONNECTIONS_ROUNDROBIN:
		lb.Logic = &LeastConnectionsRoundRobinLogic{}
	default:
		log.Error().Msgf("Balancer mode '%v' is not supported.", lb.Mode)
	}

	// Initialize Balancer logic
	if lb.Logic != nil {
		lb.Logic.Init()
	}

	lb.DeubgDataMutext = &sync.Mutex{}
}

func (lb *Balancer) IsAvailable() bool {
	return lb.State == LB_STATE_ACTIVE
}

func (lb *Balancer) _parseCustomHeaderValue(header *CustomHeader, req *http.Request) string {
	if header.Value == "[[protocol]]" {
		return req.Proto
	} else if header.Value == "[[client.host]]" {
		return req.Host
	} else if header.Value == "[[tls.version]]" {
		if req.TLS != nil {
			return fmt.Sprint(req.TLS.Version)
		}
		return ""
	} else if header.Value == "[[balancer.id]]" {
		return lb.Id
	}
	return header.Value
}

func (lb *Balancer) AddCustomHeaders(req *http.Request) {
	headersSetCounter := 0
	if len(lb.CustomHeaderRules) > 0 {
		for _, rule := range lb.CustomHeaderRules {
			if rule.Method == "any" || req.Method == rule.Method {
				for _, header := range rule.Headers {
					req.Header.Set(header.Name, lb._parseCustomHeaderValue(&header, req))
					headersSetCounter++
				}
			}
		}
	}
}

func (lb *Balancer) serveProxy(rw http.ResponseWriter, req *http.Request) error {
	target := lb.Logic.Next(lb)
	if target == nil {
		log.Info().Msg("No targets found")
		return errors.New("notfound")
	}
	log.Debug().Str("uri", req.RequestURI).Str("balancer", lb.Id).Str("to", target.Address).Msg("- Forwarding request")

	lb.liveConnections.Add(1)
	// Add Custom headers if matches any
	lb.AddCustomHeaders(req)

	isSuccessful := target.Serve(rw, req)

	if !isSuccessful {
		log.Info().Str("uri", req.RequestURI).Str("balancer", lb.Id).Str("to", target.Address).Msg("Target unreachable")
	}
	lb.liveConnections.Done()
	return nil
}

func (lb *Balancer) UpdateState() {
	if len(lb.Targets) > 0 {
		lb.State = LB_STATE_ACTIVE
	}
}

func (lb *Balancer) AddNewServer(targetConfig *TargetYAMLConfig) {
	target := NewTarget(targetConfig)
	target.MarkAsReachable()
	lb.Targets = append(lb.Targets, target)
	lb.UpdateState()
}
