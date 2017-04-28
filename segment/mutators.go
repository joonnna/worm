package main

import ()

func (s *Seg) setTargetSegments(target int) {
	s.targetMutex.Lock()
	s.targetSegments = target
	s.targetMutex.Unlock()
}

func (s *Seg) GetTargetSegments() int {
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

func (s *Seg) getNumKilled() int {
	s.numKilledMutex.RLock()
	ret := s.numKilled
	s.numKilledMutex.RUnlock()

	return ret
}

func (s *Seg) resetNumKilled() {
	s.numKilledMutex.Lock()
	s.numKilled = 0
	s.numKilledMutex.Unlock()
}

func (s *Seg) incrementNumKilled(increment int) {
	s.numKilledMutex.Lock()
	s.numKilled += increment
	s.numKilledMutex.Unlock()
}

func (s *Seg) getNumStopped() int {
	s.numStoppedMutex.RLock()
	ret := s.numStopped
	s.numStoppedMutex.RUnlock()

	return ret
}

func (s *Seg) incrementNumStopped(increment int) {
	s.numStoppedMutex.Lock()
	s.numStopped += increment
	s.numStoppedMutex.Unlock()
}

func (s *Seg) resetNumStopped() {
	s.numStoppedMutex.Lock()
	s.numStopped = 0
	s.numStoppedMutex.Unlock()
}

func (s *Seg) GetKillRate() float32 {
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
