package simulator

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"maps"
	"math"
	"math/rand"
	"slices"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/swim-timer/internal/data"
	"github.com/barthjs/swim-timer/internal/store"
)

// Simulator generates simulated timing data for a race event.
type Simulator struct {
	eventID int
	logger  zerolog.Logger
	store   *store.RaceDataStore
	writer  io.Writer
}

// New creates a new Simulator instance.
func New(
	eventID int,
	logger zerolog.Logger,
	store *store.RaceDataStore,
	writer io.Writer,
) *Simulator {
	return &Simulator{
		eventID: eventID,
		logger:  logger.With().Str("component", "simulator").Logger(),
		store:   store,
		writer:  writer,
	}
}

// Run starts the simulation loop.
func (s *Simulator) Run(ctx context.Context) {
	starts := s.store.GetStarts()
	if len(starts) == 0 {
		s.logger.Error().Msg("no starts found in store")
		return
	}

	// Group starts by event and heat
	heatsMap := make(map[int][]*data.StartEntry)
	for _, start := range starts {
		if start.Event == s.eventID {
			heatsMap[start.Heat] = append(heatsMap[start.Heat], start)
		}
	}

	if len(heatsMap) == 0 {
		s.logger.Error().Int("eventID", s.eventID).Msg("no starts found for event in store")
		return
	}

	// Sort heat numbers
	heatNums := slices.Collect(maps.Keys(heatsMap))
	slices.Sort(heatNums)

	for {
		for _, hNum := range heatNums {
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
				s.simulateHeat(ctx, s.eventID, hNum+1, heatsMap[hNum])
			}
		}
	}
}

// simulateHeat runs a single heat, sending time updates for each participant.
func (s *Simulator) simulateHeat(ctx context.Context, eventNum, heatNum int, entries []*data.StartEntry) {
	s.logger.Info().Int("event", eventNum).Int("heat", heatNum).Msg("Simulating heat")

	// Filter entries to have only one per lane
	laneMap := make(map[int]*data.StartEntry)
	for _, entry := range entries {
		if _, ok := laneMap[entry.Lane]; !ok {
			laneMap[entry.Lane] = entry
		}
	}

	filteredEntries := slices.Collect(maps.Values(laneMap))

	// Determine min/max lane for middle speed calculation
	minLane, maxLane := len(filteredEntries), 0
	for _, entry := range filteredEntries {
		if entry.Lane < minLane {
			minLane = entry.Lane
		}
		if entry.Lane > maxLane {
			maxLane = entry.Lane
		}
	}
	middleLane := float64(minLane+maxLane) / 2.0

	lapCount := 2 // Default to 2 laps (100m)
	if race, ok := s.store.GetRace(eventNum); ok && race.Distance != nil && race.Distance.Length > 0 {
		lapCount = race.Distance.Length / 50
		if lapCount == 0 {
			lapCount = 1
		}
	}

	// Load frame (Ready at start)
	s.sendFrame('0', ' ', lapCount, eventNum, heatNum, 0, 0, 0, 0)
	time.Sleep(10 * time.Second)

	// Start frame
	s.sendFrame('2', 'S', lapCount, eventNum, heatNum, 0, 0, 0, 0)
	startTime := time.Now()

	type participant struct {
		entry *data.StartEntry
		speed time.Duration // time per 50m
		time  time.Duration
		rank  int
	}

	parts := make([]*participant, len(filteredEntries))
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i, entry := range filteredEntries {
		// Base speed between 20 s and 30 s per 50m
		baseSpeed := 20*time.Second + time.Duration(rng.Intn(10000))*time.Millisecond
		// Penalty for being far from the middle
		distFromMiddle := math.Abs(float64(entry.Lane) - middleLane)
		penalty := time.Duration(distFromMiddle * float64(500*time.Millisecond))

		parts[i] = &participant{
			entry: entry,
			speed: baseSpeed + penalty,
		}
	}

	// Simulate all laps
	for lap := 1; lap <= lapCount; lap++ {
		// Assign times for this lap
		for _, p := range parts {
			p.time = p.speed * time.Duration(lap)
		}

		// Sort by time to assign rank
		sortedParts := slices.Clone(parts)
		slices.SortFunc(sortedParts, func(a, b *participant) int {
			return cmp.Compare(a.time, b.time)
		})

		for i, p := range sortedParts {
			p.rank = i + 1
		}

		// Wait and send frames in order of arrival
		for _, p := range sortedParts {
			remaining := p.time - time.Since(startTime)
			if remaining > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(remaining):
				}
			}

			kind := byte('I')
			if lap == lapCount {
				kind = 'A'
			}
			s.sendFrame('2', kind, lapCount, eventNum, heatNum, p.rank, p.entry.Lane, lap, p.time)
		}
	}

	// Official frame
	s.sendFrame('1', ' ', lapCount, eventNum, heatNum, 0, 0, 0, 0)
}

// sendFrame constructs and writes timing frames using the omega transmission protocol.
func (s *Simulator) sendFrame(msgType, kind byte, totalLaps, event, heat, rank, lane, lap int, t time.Duration) {
	// Part 1: [SOH][STX][HOME]ABCDDEEFFFGG¬¬HH[EOT] (20 bytes)
	p1 := make([]byte, 20)
	p1[0] = 0x01
	p1[1] = 0x02
	p1[2] = 0x08
	p1[3] = msgType
	p1[4] = kind
	p1[5] = 0x20
	p1[6] = 0x20
	p1[7] = 0x20
	copy(p1[8:10], fmt.Sprintf("%02d", totalLaps))
	copy(p1[10:13], fmt.Sprintf("%03d", event))
	copy(p1[13:15], fmt.Sprintf("%02d", heat))
	p1[15] = 0x20
	p1[16] = 0x20
	if rank > 0 {
		copy(p1[17:19], fmt.Sprintf("%2d", rank))
	} else {
		p1[17] = 0x20
		p1[18] = 0x20
	}
	p1[19] = 0x04

	// Part 2: [SOH][STX][HOME][LF]JKK[STX]Hh:Mm:Ss.dc¬[EOT] (21 bytes)
	p2 := make([]byte, 21)
	p2[0] = 0x01
	p2[1] = 0x02
	p2[2] = 0x08
	p2[3] = 0x10
	if lane > 0 {
		p2[4] = byte('0' + lane%10)
	} else {
		p2[4] = 0x20
	}
	copy(p2[5:7], fmt.Sprintf("%02d", lap))
	p2[7] = 0x02

	h := int(t.Hours())
	m := int(t.Minutes()) % 60
	secs := int(t.Seconds()) % 60
	dc := (int(t.Milliseconds()) % 1000) / 10
	copy(p2[8:19], fmt.Sprintf("%02d:%02d:%02d.%02d", h, m, secs, dc))
	p2[19] = 0x20
	p2[20] = 0x04

	_, _ = s.writer.Write(p1)
	_, _ = s.writer.Write(p2)
}
