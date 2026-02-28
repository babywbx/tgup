package app

import (
	"math/rand"
	"time"
)

// Clock abstracts time for deterministic tests.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// RNG abstracts random sampling for retry/backoff decisions.
type RNG interface {
	Float64() float64
}

// SystemClock is the production Clock implementation.
type SystemClock struct{}

// Now implements Clock.
func (SystemClock) Now() time.Time { return time.Now() }

// Sleep implements Clock.
func (SystemClock) Sleep(d time.Duration) { time.Sleep(d) }

// SystemRNG is the production RNG implementation.
type SystemRNG struct{}

// Float64 implements RNG.
func (SystemRNG) Float64() float64 { return rand.Float64() }
