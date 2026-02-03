package dns

import "log"

type AdBlocker struct {
	Port string
}

func NewAdBlocker(port string) *AdBlocker {
	return &AdBlocker{Port: port}
}

func (a *AdBlocker) Start() error {
	log.Printf("[DNS] Starting AdBlocker on %s (Skeleton)", a.Port)
	select {} 
}