package src

import (
	"math/rand"
	"sync"
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
	var successTarget = make(chan *Target, 1)
	go func() {
		liveTargets := []int{}
		for index, target := range lb.Targets {
			if target.IsAlive() {
				liveTargets = append(liveTargets, index)
			}
		}
		liveTargetsLength := len(liveTargets)
		if liveTargetsLength == 0 {
			successTarget <- nil
			return
		}

		randomIndex := rand.Intn(liveTargetsLength)
		successTarget <- lb.Targets[liveTargets[randomIndex]]
	}()

	select {
	case <-time.After(lb.TargetWaitTimeout):
		log.Info().
			Str("balancer", lb.Id).
			Str("mode", lb.Mode).
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
	breakerFlag := false
	go func(breakerFlag *bool) {
		for i := 0; i < targetCount; i++ {
			target := lb.Targets[rbl.Counter%targetCount]
			rbl.Counter++
			if target.IsAlive() {
				rbl.Counter = rbl.Counter % targetCount
				successTarget <- target
				return
			}
			if *breakerFlag {
				return
			}
		}
	}(&breakerFlag)

	select {
	case <-time.After(lb.TargetWaitTimeout):
		log.Info().
			Str("balancer", lb.Id).
			Str("mode", lb.Mode).
			Msg("Request is timing out due to no available targets.")
		breakerFlag = true
		return nil
	case target := <-successTarget:
		return target
	}
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

	successTarget := make(chan *Target, 1)
	breakerFlag := false
	go func(breakerFlag *bool) {
		for i := 0; i < targetCount; i++ {
			target := lb.Targets[wrbl.Counter%targetCount]
			wrbl.WeightCounter++
			if target.Weight <= wrbl.WeightCounter || !target.IsAlive() {
				wrbl.Counter++
				wrbl.WeightCounter = 0
			}
			if target.IsAlive() {
				wrbl.Counter %= targetCount
				successTarget <- target
				return
			}
			if *breakerFlag {
				return
			}
		}
	}(&breakerFlag)

	select {
	case <-time.After(lb.TargetWaitTimeout):
		log.Info().
			Str("balancer", lb.Id).
			Str("mode", lb.Mode).
			Msg("Request is timing out due to no available targets.")
		breakerFlag = true
		return nil
	case target := <-successTarget:
		return target
	}
}

/******* Least Connections Logic ********/

type LeastConnectionsRandomLogic struct {
}

func (lc *LeastConnectionsRandomLogic) Init() {

}

func (lc *LeastConnectionsRandomLogic) Next(lb *Balancer) *Target {
	var successTarget = make(chan *Target, 1)
	go func() {
		pool := []*Target{}

		minTarget := lb.Targets[0]
		pool = append(pool, minTarget)

		for _, nextTarget := range lb.Targets[1:] {
			if nextTarget.Connections < minTarget.Connections {
				minTarget = nextTarget
				pool = []*Target{
					minTarget,
				}
			} else if nextTarget.Connections == minTarget.Connections {
				pool = append(pool, nextTarget)
			}
		}
		poolSize := len(pool)
		if poolSize > 1 {
			randIndex := rand.Intn(poolSize)
			minTarget = pool[randIndex]
		}
		successTarget <- minTarget
	}()

	select {
	case <-time.After(lb.TargetWaitTimeout):
		log.Info().Str("balancer", lb.Id).Str("mode", lb.Mode).Msg("Request is timing out due to no available targets.")
		return nil
	case target := <-successTarget:
		return target
	}
}

/******* Least Connections RoundRobin Logic ********/

type LeastConnectionsRoundRobinLogic struct {
	mu        sync.Mutex
	LastIndex int
}

func (lc *LeastConnectionsRoundRobinLogic) Init() {
	lc.LastIndex = -1
}

func (lc *LeastConnectionsRoundRobinLogic) Next(lb *Balancer) *Target {
	var successTarget = make(chan *Target, 1)
	go func() {
		var indexPool []int
		var minTarget *Target

		lc.mu.Lock()
		defer lc.mu.Unlock()

		for index, nextTarget := range lb.Targets {
			if nextTarget.IsAlive() {
				if minTarget == nil || nextTarget.Connections < minTarget.Connections {
					minTarget = nextTarget
					indexPool = []int{index}
				} else if nextTarget.Connections == minTarget.Connections {
					indexPool = append(indexPool, index)
				}
			}
		}
		candidateTargetIndex := -1
		if len(indexPool) > 0 {
			for _, index := range indexPool {
				if candidateTargetIndex == -1 {
					candidateTargetIndex = index
				}
				if index > lc.LastIndex {
					candidateTargetIndex = index
					break
				}
			}
			lc.LastIndex = candidateTargetIndex
			successTarget <- lb.Targets[candidateTargetIndex]
			return
		}
	}()

	select {
	case <-time.After(lb.TargetWaitTimeout):
		log.Info().Str("balancer", lb.Id).Str("mode", lb.Mode).Msg("Request is timing out due to no available targets.")
		return nil
	case target := <-successTarget:
		return target
	}
}
