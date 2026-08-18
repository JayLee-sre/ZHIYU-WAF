// Package behavior provides bounded, local request behavior aggregation.
package behavior

import (
	"context"
	"net"
	"sort"
	"sync"
	"time"
)

// EventType identifies behavior counters used by the V2 risk engine.
type EventType string

const (
	EventRequest      EventType = "request"
	EventNotFound     EventType = "not_found"
	EventAttack       EventType = "attack"
	EventScanner      EventType = "scanner"
	EventLoginFailure EventType = "login_failure"
)

// Event is a minimal, privacy-preserving behavior observation.
type Event struct {
	IP        net.IP
	Type      EventType
	CreatedAt time.Time
}

// Snapshot represents counts in the configured time windows.
type Snapshot struct {
	IP      net.IP         `json:"ip"`
	Windows map[string]Map `json:"windows"`
}

// Map stores one counter per behavior event type.
type Map map[EventType]int

// Config bounds retained memory and controls deterministic cleanup.
type Config struct {
	MaxIPs       int
	Retention    time.Duration
	Windows      []time.Duration
	CleanupEvery time.Duration
}

type bucket struct {
	events   []Event
	lastSeen time.Time
}

// Engine is safe for concurrent observation. It uses an LRU-like eviction by
// oldest last-seen value rather than an unbounded map.
type Engine struct {
	config Config
	mu     sync.Mutex
	ips    map[string]*bucket
	stop   chan struct{}
	done   chan struct{}
}

func New(config Config) *Engine {
	if config.MaxIPs <= 0 {
		config.MaxIPs = 10000
	}
	if config.Retention <= 0 {
		config.Retention = time.Hour
	}
	if len(config.Windows) == 0 {
		config.Windows = []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, time.Hour}
	}
	if config.CleanupEvery <= 0 {
		config.CleanupEvery = time.Minute
	}
	engine := &Engine{config: config, ips: make(map[string]*bucket), stop: make(chan struct{}), done: make(chan struct{})}
	go engine.cleanupLoop()
	return engine
}

func (e *Engine) Close() {
	if e == nil {
		return
	}
	select {
	case <-e.stop:
		return
	default:
		close(e.stop)
		<-e.done
	}
}

// Observe adds an event after validating that it contains a canonical IP.
func (e *Engine) Observe(_ context.Context, event Event) {
	if e == nil || event.IP == nil {
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	key := canonicalIP(event.IP)
	if key == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.ips[key]; !ok && len(e.ips) >= e.config.MaxIPs {
		e.evictOldestLocked()
	}
	entry := e.ips[key]
	if entry == nil {
		entry = &bucket{}
		e.ips[key] = entry
	}
	entry.events = append(entry.events, event)
	entry.lastSeen = event.CreatedAt
	e.trimLocked(entry, event.CreatedAt)
}

// Snapshot returns a stable copy of the counts. It does not retain references
// to internal state.
func (e *Engine) Snapshot(_ context.Context, ip net.IP) Snapshot {
	result := Snapshot{IP: append(net.IP(nil), ip...), Windows: make(map[string]Map)}
	if e == nil {
		return result
	}
	key := canonicalIP(ip)
	if key == "" {
		return result
	}
	now := time.Now()
	e.mu.Lock()
	defer e.mu.Unlock()
	entry := e.ips[key]
	if entry == nil {
		return result
	}
	e.trimLocked(entry, now)
	for _, window := range e.config.Windows {
		counts := make(Map)
		cutoff := now.Add(-window)
		for _, event := range entry.events {
			if !event.CreatedAt.Before(cutoff) {
				counts[event.Type]++
			}
		}
		result.Windows[window.String()] = counts
	}
	return result
}

func (e *Engine) cleanupLoop() {
	ticker := time.NewTicker(e.config.CleanupEvery)
	defer ticker.Stop()
	defer close(e.done)
	for {
		select {
		case <-ticker.C:
			e.cleanup(time.Now())
		case <-e.stop:
			return
		}
	}
}

func (e *Engine) cleanup(now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for key, entry := range e.ips {
		e.trimLocked(entry, now)
		if len(entry.events) == 0 && now.Sub(entry.lastSeen) > e.config.Retention {
			delete(e.ips, key)
		}
	}
}

func (e *Engine) trimLocked(entry *bucket, now time.Time) {
	cutoff := now.Add(-e.config.Retention)
	first := sort.Search(len(entry.events), func(i int) bool { return !entry.events[i].CreatedAt.Before(cutoff) })
	if first > 0 {
		entry.events = append([]Event(nil), entry.events[first:]...)
	}
}

func (e *Engine) evictOldestLocked() {
	var oldestKey string
	var oldest time.Time
	for key, entry := range e.ips {
		if oldestKey == "" || entry.lastSeen.Before(oldest) {
			oldestKey, oldest = key, entry.lastSeen
		}
	}
	if oldestKey != "" {
		delete(e.ips, oldestKey)
	}
}

func canonicalIP(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	if v6 := ip.To16(); v6 != nil {
		return v6.String()
	}
	return ""
}
