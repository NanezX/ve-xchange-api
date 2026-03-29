package main

import (
	"sync"
)

type State struct {
	sync.RWMutex
	Rates ExchageRates
}

var AppState = &State{}

func (s *State) UpdateRates(newRates ExchageRates) {
	s.Lock()
	defer s.Unlock()
	s.Rates = newRates
}

func (s *State) GetRates() ExchageRates {
	s.RLock()
	defer s.RUnlock()
	return s.Rates
}
