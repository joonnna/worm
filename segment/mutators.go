package main

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
