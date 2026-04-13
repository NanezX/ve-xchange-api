package main

import (
	"sync"
)

type State struct {
	sync.RWMutex
	Rates ExchangeRates
}

var AppState = &State{}

func (s *State) UpdateRates(newRates ExchangeRates) {
	s.Lock()
	defer s.Unlock()
	s.Rates = newRates
}

func (s *State) GetRates() ExchangeRates {
	s.RLock()
	defer s.RUnlock()
	return s.Rates
}
