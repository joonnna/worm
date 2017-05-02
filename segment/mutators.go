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

func (s *Seg) incrementNumAdded() {
	s.numAddedMutex.Lock()
	s.numAdded = s.numAdded + 1
	s.numAddedMutex.Unlock()
}

func (s *Seg) getNumAdded() int {
	s.numAddedMutex.RLock()
	ret := s.numAdded
	s.numAddedMutex.RUnlock()

	return ret
}

func (s *Seg) resetNumAdded() {
	s.numAddedMutex.Lock()
	s.numAdded = 0
	s.numAddedMutex.Unlock()
}

func (s *Seg) setAddMap(key string, value int) {
	s.addMapMutex.Lock()
	s.addMap[key] = value
	s.addMapMutex.Unlock()
}

func (s *Seg) calcAdded() int {
	s.addMapMutex.Lock()

	total := 0
	for _, val := range s.addMap {
		total += val
	}

	s.addMapMutex.Unlock()

	return total
}

func (s *Seg) resetAddMap() {
	s.addMapMutex.Lock()

	for key, _ := range s.addMap {
		s.addMap[key] = 0
	}

	s.addMapMutex.Unlock()

}
