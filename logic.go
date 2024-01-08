package main

import (
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

type BalancerLogic interface {
	Next(lb *Balancer) *Target
	Init()
}

/****** Random Logic ******/
type RandomLogic struct{}

func (rl *RandomLogic) Init() {

}

func (rl *RandomLogic) Next(lb *Balancer) *Target {
	targetsLength := len(lb.Targets)
	randomIndex := rand.Intn(targetsLength)

	var successTarget = make(chan *Target, 1)
	go func() {
		for {
			target := lb.Targets[randomIndex]
			if target.IsAlive() {
				successTarget <- target
			}
		}
	}()

	select {
	case <-time.After(lb.TargetWaitTimeout):
		log.Info().
			Str("balancer", lb.Id).
			Str("mode", LB_MODE_RANDOM).
			Msg("Request is timing out due to no available targets.")
		return nil
	case target := <-successTarget:
		return target
	}
}

/****** Round Robin *******/
type RoundRobinLogic struct {
	Counter int
}

func (rbl *RoundRobinLogic) Init() {
	rbl.Counter = 0
}

func (rbl *RoundRobinLogic) Next(lb *Balancer) *Target {
	targetCount := len(lb.Targets)

	var successTarget = make(chan *Target, 1)
	go func() {
		for {
			target := lb.Targets[rbl.Counter%targetCount]
			rbl.Counter++
			if target.IsAlive() {
				rbl.Counter = rbl.Counter % targetCount
				successTarget <- target
				break
			}
		}
	}()

	select {
	case <-time.After(lb.TargetWaitTimeout):
		log.Info().
			Str("balancer", lb.Id).
			Str("mode", LB_MODE_ROUNDROBIN).
			Msg("Request is timing out due to no available targets.")
		return nil
	case target := <-successTarget:
		return target
	}
	return nil
}

/****** Weighted Round Robin *******/
type WeightedRoundRobinLogic struct {
	Counter       int
	WeightCounter int
}

func (wrbl *WeightedRoundRobinLogic) Init() {
	wrbl.Counter = 0
	wrbl.WeightCounter = 0
}

func (wrbl *WeightedRoundRobinLogic) Next(lb *Balancer) *Target {
	targetCount := len(lb.Targets)

	var successTarget = make(chan *Target, 1)
	go func() {
		for {
			target := lb.Targets[wrbl.Counter%targetCount]
			wrbl.WeightCounter++
			if target.Weight <= wrbl.WeightCounter {
				wrbl.Counter++
				wrbl.WeightCounter = 0
			}
			if target.IsAlive() {
				wrbl.Counter = wrbl.Counter % targetCount
				successTarget <- target
				break
			}
		}
	}()

	select {
	case <-time.After(lb.TargetWaitTimeout):
		log.Info().
			Str("balancer", lb.Id).
			Str("mode", LB_MODE_WEIGHTED_ROUNDROBIN).
			Msg("Request is timing out due to no available targets.")
		return nil
	case target := <-successTarget:
		return target
	}
}
