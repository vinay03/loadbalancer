package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type LB_STATE int

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
}

var LoadBalancersPool map[string]*Balancer

type BalancerLogic interface {
	Next(lb *Balancer) *Target
}

type RoundRobinLogic struct {
	Counter int
}

func (rbl *RoundRobinLogic) Next(lb *Balancer) *Target {
	targetCount := len(lb.Targets)

	ctx, cancelFunc := context.WithTimeout(context.Background(), lb.TargetWaitTimeout)
	defer cancelFunc()

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Str("id", lb.Id).
				Msg("Request is timing out due to no available targets.")
			return nil
		default:
			target := lb.Targets[rbl.Counter%targetCount]
			rbl.Counter++
			if target.IsAlive() {
				rbl.Counter = rbl.Counter % targetCount
				return target
			}
		}
	}
}

func (lb *Balancer) SetBalancerLogic() {
	if lb.Mode == LB_MODE_ROUNDROBIN {
		lb.Logic = &RoundRobinLogic{}
	}
}

// func (lb *Balancer) getNextAvailableServer() *Target {
// 	target := lb.Targets[lb.roundRobinCount%len(lb.Targets)]
// 	for !target.IsAlive() {
// 		lb.roundRobinCount++
// 		target = lb.Targets[lb.roundRobinCount%len(lb.Targets)]
// 	}
// 	lb.roundRobinCount++
// 	return target
// }

// http.Handler is a interface that expects ServeHTTP() function to be implemented.
// func (lb *Balancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
// 	lb.serveProxy(rw, req)
// }

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
	if len(lb.CustomHeaderRules) > 0 {
		for _, rule := range lb.CustomHeaderRules {
			if rule.Method == "any" || req.Method == rule.Method {
				for _, header := range rule.Headers {
					req.Header.Set(header.Name, lb._parseCustomHeaderValue(&header, req))
				}
			}
		}
	}
}

func (lb *Balancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	// target := lb.NextAvailableServer(lb)
	target := lb.Logic.Next(lb)
	if target == nil {
		return
	}
	log.Debug().
		Str("uri", req.RequestURI).
		Str("balancer", lb.Id).
		Str("to", target.Address()).
		Msg("Forwarding request")

	lb.liveConnections.Add(1)
	// Add Custom headers if matches any
	lb.AddCustomHeaders(req)

	target.Serve(rw, req)
	lb.liveConnections.Done()
}

func (lb *Balancer) UpdateState() {
	if len(lb.Targets) > 0 {
		lb.State = LB_STATE_ACTIVE
	}
}

func (lb *Balancer) AddNewServer(addr string) {
	lb.Targets = append(lb.Targets, NewTarget(addr))
	lb.UpdateState()
}
