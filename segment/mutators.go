package main

import (
	"time"
)

func (s *Seg) setTargetSegments(target int) {
	s.targetMutex.Lock()
	s.targetSegments = target
	s.targetMutex.Unlock()
}

func (s *Seg) getTargetSegments() int {
	s.targetMutex.RLock()
	ret := s.targetSegments
	s.targetMutex.RUnlock()

	return ret
}

func (s *Seg) setLeader(host string) {
	s.leaderMutex.Lock()
	s.currentLeader = host
	s.leaderMutex.Unlock()
}

func (s *Seg) getLeader() string {
	s.leaderMutex.RLock()
	ret := s.currentLeader
	s.leaderMutex.RUnlock()

	return ret
}

func (s *Seg) getKillRate() float32 {
	s.killRateMutex.RLock()
	ret := s.killRate
	s.killRateMutex.RUnlock()

	return ret
}

func (s *Seg) setKillRate(newKillRate float32) {
	s.killRateMutex.Lock()
	s.killRate = newKillRate
	s.killRateMutex.Unlock()
}

func (s *Seg) getPending(key string) bool {
	s.pendingMutex.RLock()
	_, ok := s.pendingMap[key]
	s.pendingMutex.RUnlock()

	return ok
}

func (s *Seg) deletePending(key string) {
	s.pendingMutex.Lock()
	delete(s.pendingMap, key)
	s.pendingMutex.Unlock()
}

func (s *Seg) setPending(key string) {
	s.pendingMutex.Lock()
	s.pendingMap[key] = time.Now()
	s.pendingMutex.Unlock()
}

func (s *Seg) lenPending() int {
	s.pendingMutex.RLock()
	ret := len(s.pendingMap)
	s.pendingMutex.RUnlock()

	return ret
}
