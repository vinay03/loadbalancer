package main

import (
	"context"

	"github.com/rs/zerolog/log"
)

type BalancerLogic interface {
	Next(lb *Balancer) *Target
	Init()
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

	ctx, cancelFunc := context.WithTimeout(context.Background(), lb.TargetWaitTimeout)
	defer cancelFunc()

	var successTarget = make(chan *Target, 1)

	go func() {
		for {
			target := lb.Targets[rbl.Counter%targetCount]
			rbl.Counter++
			if target.IsAlive() {
				rbl.Counter = rbl.Counter % targetCount
				log.Info().Msg("Target chosen")
				successTarget <- target
				break
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Str("id", lb.Id).
				Msg("Request is timing out due to no available targets.")
			return nil
		case target := <-successTarget:
			log.Info().Msg("Target accepted")
			return target
			// default:
			// target := lb.Targets[wrbl.Counter%targetCount]
			// wrbl.Counter++
			// if target.IsAlive() {
			// 	wrbl.Counter = wrbl.Counter % targetCount
			// 	return target
			// }
		}
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
			target := lb.Targets[wrbl.Counter%targetCount]
			wrbl.Counter++
			if target.IsAlive() {
				wrbl.Counter = wrbl.Counter % targetCount
				return target
			}
		}
	}
}
