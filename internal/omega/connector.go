package omega

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"

	"github.com/barthjs/swim-timer/internal/data"
)

// StateManager provides access to the shared race state.
type StateManager interface {
	GetState() *data.RaceState
	UpdateState(state *data.RaceState)
}

// RaceDataProvider provides read access to race and start list data.
type RaceDataProvider interface {
	GetRace(event int) (*data.Race, bool)
	GetStarts() []*data.StartEntry
	GetStartsByEventHeat(event, heat int) []*data.StartEntry
}

type logEntry struct {
	timestamp   time.Time
	msgType     string
	msgKind     string
	timeType    string
	usedLanes   string
	lapCount    string
	event       string
	heat        string
	rank        string
	lane        string
	lap         string
	time        string
	omegaTimeMS string
	localDayMS  string
	clockOffset string
	delay       string
	hexData     string
}

// portConnection represents either a serial port or TCP connection
type portConnection interface {
	Read(b []byte) (n int, err error)
	Close() error
}

type Connector struct {
	poolLength  int
	serialPort  string
	raceDataDir string
	logger      zerolog.Logger
	store       RaceDataProvider
	hub         StateManager
	logChannel  chan logEntry
	// clockOffset is the baseline difference between the local clock and the
	// Omega device clock, established on the first received start signal.
	// It is used to correct the StartTime when a start signal arrives late.
	clockOffset time.Duration
	clockSynced bool
}

// NewConnector creates a new Connector instance to handle
// communication with the Omega timing device.
func NewConnector(
	poolLength int,
	serialPort string,
	raceDataDir string,
	logger zerolog.Logger,
	store RaceDataProvider,
	hub StateManager,
) *Connector {
	return &Connector{
		poolLength:  poolLength,
		serialPort:  serialPort,
		raceDataDir: raceDataDir,
		logger:      logger,
		store:       store,
		hub:         hub,
	}
}

// Run starts the connector and continuously attempts to connect to an available serial port,
// then reads and processes timing frames from the Omega device until context is canceled.
func (c *Connector) Run(ctx context.Context) {
	if c.logger.GetLevel() == zerolog.DebugLevel {
		c.logChannel = make(chan logEntry, 100)
		go c.loggerWorker(c.logChannel)

		defer func() {
			close(c.logChannel)
		}()
	}

	var availablePorts []string

	for {
		if c.serialPort != "" {
			availablePorts = []string{c.serialPort}
			break
		}

		ports, err := enumerator.GetDetailedPortsList()
		if err != nil {
			c.logger.Error().Err(err).Msg("failed to list serial ports")
		} else if len(ports) == 0 {
			c.logger.Warn().Msg("no serial ports found, retrying in 5s")
		} else {
			for _, p := range ports {
				availablePorts = append(availablePorts, p.Name)
			}
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}

	mode := &serial.Mode{
		BaudRate: 9600,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			c.connectAndRead(ctx, availablePorts, mode)
			time.Sleep(5 * time.Second)
		}
	}
}

// connectAndRead attempts to open a serial port connection and read timing frames.
// It tries each port in sequence until a successful connection is established,
// then continuously reads the data and assembles two-part frames before processing them.
func (c *Connector) connectAndRead(ctx context.Context, ports []string, mode *serial.Mode) {
	var port portConnection
	var err error

	for _, portName := range ports {
		select {
		case <-ctx.Done():
			return
		default:
		}

		port, err = c.openConnection(portName, mode)
		if err == nil {
			c.serialPort = portName
			c.logger.Info().Str("port", portName).Msg("connected to serial port")
			break
		} else {
			c.logger.Debug().Err(err).Str("port", portName).Msg("port busy or cannot open")
		}
	}

	if port == nil {
		c.logger.Error().Msg("no available serial port, retrying in 5 seconds")
		return
	}
	defer func() {
		_ = port.Close()
	}()

	reader := bufio.NewReader(port)
	var firstLine []byte

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// EOT separates the split transmission blocks from the Omega system
			line, err := reader.ReadBytes(byteEOT)
			if err != nil {
				c.logger.Error().Err(err).Msg("error reading from serial port")
				return
			}

			if len(line) == framePart1Size {
				firstLine = make([]byte, len(line))
				copy(firstLine, line)
			} else if len(line) == framePart2Size && len(firstLine) == framePart1Size {
				combined := append(firstLine, line...)
				c.processLine(combined)
				firstLine = nil
			} else {
				firstLine = nil
			}
		}
	}
}

// openConnection opens either a serial port or TCP connection depending on the port name format
func (c *Connector) openConnection(portName string, mode *serial.Mode) (portConnection, error) {
	if strings.Contains(portName, ":") {
		conn, err := net.Dial("tcp", portName)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
	return serial.Open(portName, mode)
}

// processLine parses a complete timing frame from the Omega transmission protocol
// and updates the race state. It validates the frame structure, extracts lap count,
// event, heat, and message type to determine the appropriate action.
func (c *Connector) processLine(line []byte) {
	if len(line) != frameTotalSize {
		return
	}

	// SOH, STX, HOME are mandatory header bytes for valid timing frames
	if line[offsetSOH] != byteSOH || line[offsetSTX] != byteSTX || line[offsetHOME] != byteHOME {
		return
	}

	if c.logger.GetLevel() == zerolog.DebugLevel {
		c.logFrame(string(line))
	}

	lapCount, err := strconv.Atoi(strings.TrimSpace(string(line[offsetLapCount:offsetLapCountEnd])))
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to parse lap count")
		return
	}
	event, err := strconv.Atoi(strings.TrimSpace(string(line[offsetEvent:offsetEventEnd])))
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to parse event")
		return
	}
	heat, err := strconv.Atoi(strings.TrimSpace(string(line[offsetHeat:offsetHeatEnd])))
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to parse heat")
		return
	}

	msgType, msgKind := line[offsetMsgType], line[offsetMsgKind]

	switch {
	case msgType == msgTypeLoad && msgKind == msgKindLoad:
		c.updateStatus(event, heat, lapCount, data.RaceStatusLoad, nil)
		return
	case msgType == msgTypeOfficial && msgKind == msgKindLoad:
		c.updateStatus(event, heat, lapCount, data.RaceStatusOfficial, nil)
		return
	case msgType == msgTypeRunning && msgKind == msgKindStart:
		c.updateStatus(event, heat, lapCount, data.RaceStatusStart, line)
		return
	case msgType == msgTypeRunning && msgKind == msgKindIntermediate:
		c.updateTime(event, heat, lapCount, line, false)
		return
	case msgType == msgTypeRunning && msgKind == msgKindFinish:
		c.updateTime(event, heat, lapCount, line, true)
		return
	default:
		return
	}
}

// updateStatus updates the race status for an event and heat,
// creating a new race state if necessary. It initializes lanes with competitor information
// from the start list and broadcasts the state change to connected clients.
// For start signals, the raw frame is passed to extract the Omega device time-of-day for clock synchronization.
func (c *Connector) updateStatus(event, heat, lapCount int, status data.RaceStatus, frame []byte) {
	state := c.hub.GetState()

	// If it's a new event/heat, we create a new state
	if state == nil || state.Event != event || state.Heat != heat {
		race, ok := c.store.GetRace(event)
		if !ok {
			c.logger.Error().Int("event", event).Msg("race not found in store")
			return
		}

		distanceName := ""
		if race.Distance != nil {
			distanceName = race.Distance.Name
		}
		styleName := ""
		if race.Style != nil {
			styleName = race.Style.Name
		}
		categoryName := ""
		if race.Category != nil {
			categoryName = race.Category.Name
		}

		state = &data.RaceState{
			Event:    event,
			Heat:     heat,
			MaxHeat:  race.NBHeat,
			LapCount: lapCount,
			Name:     strings.TrimSpace(fmt.Sprintf("%s %s %s", distanceName, styleName, categoryName)),
			Status:   status,
			Lanes:    make(map[int]*data.Lane),
		}

		if status == data.RaceStatusStart {
			state.StartTime = new(c.correctedStartTime(frame))
		}

		// The Omega protocol transmits 1-based heat numbers, while the CSV
		// start list (LSTSTART) uses 0-based heat indices, so we subtract 1.
		for _, start := range c.store.GetStartsByEventHeat(event, heat-1) {
			lane := &data.Lane{
				Number: start.Lane,
			}
			if start.Person != nil {
				lane.FirstName = start.Person.FirstName
				lane.LastName = start.Person.LastName
				lane.Birthyear = start.Person.BirthYear
				lane.Club = start.Person.GetClub()
			}
			state.Lanes[lane.Number] = lane
		}
	} else {
		state.LapCount = lapCount
		state.Status = status
		if status == data.RaceStatusStart {
			state.StartTime = new(c.correctedStartTime(frame))
		}
	}

	c.hub.UpdateState(state)
}

// updateTime extracts timing data from a frame and updates the lane information in the current race state.
// It updates the lap number, distance covered, elapsed time, and rank for the specified lane.
// The finished flag indicates whether this is an intermediate split or the final finish time.
func (c *Connector) updateTime(event, heat, lapCount int, line []byte, finished bool) {
	state := c.hub.GetState()
	if state == nil || state.Event != event || state.Heat != heat {
		return
	}

	state.LapCount = lapCount
	laneNum, _ := strconv.Atoi(strings.TrimSpace(string(line[offsetLane:offsetLaneEnd])))

	if lane, ok := state.Lanes[laneNum]; ok {
		lap, _ := strconv.Atoi(strings.TrimSpace(string(line[offsetLap:offsetLapEnd])))

		lane.Lap = lap
		lane.Meter = lap * c.getPoolLength(event)
		lane.Time = convertTimeToMS(string(line[offsetTime:offsetTimeEnd]))
		lane.Rank, _ = strconv.Atoi(strings.TrimSpace(string(line[offsetRank:offsetRankEnd])))
		if finished {
			lane.Finished = true
		}
	}

	c.hub.UpdateState(state)
}

// correctedStartTime calculates the actual race start time by comparing the Omega device's
// time-of-day (from the start frame) with the local clock. On the first start signal, the
// offset between both clocks is stored as a baseline. On later starts that may arrive
// late, this baseline is used to determine the transmission delay and shift the start time back.
func (c *Connector) correctedStartTime(frame []byte) time.Time {
	now := time.Now()

	if len(frame) < offsetTimeEnd {
		return now
	}

	omegaTimeMS := convertTimeToMS(string(frame[offsetTime:offsetTimeEnd]))
	if omegaTimeMS <= 0 {
		return now
	}

	localDayMS := localTimeOfDayMS(now)
	currentOffset := time.Duration(localDayMS-omegaTimeMS) * time.Millisecond

	if !c.clockSynced {
		c.clockOffset = currentOffset
		c.clockSynced = true
		c.logger.Info().
			Int("omega_time_ms", omegaTimeMS).
			Int("local_day_ms", localDayMS).
			Dur("clock_offset", c.clockOffset).
			Msg("clock sync established")
		return now
	}

	delay := currentOffset - c.clockOffset
	if delay < 0 {
		delay = 0
	}

	c.logger.Info().
		Dur("delay", delay).
		Dur("clock_offset", c.clockOffset).
		Dur("current_offset", currentOffset).
		Msg("start signal received with clock correction")

	return now.Add(-delay)
}

// getPoolLength returns the pool length for a given event.
// It first uses the value in the config value if available.
// Otherwise, it derives the length from race distance and lap count.
// Falls back to 50m if required data is missing or invalid.
func (c *Connector) getPoolLength(event int) int {
	if c.poolLength > 0 {
		return c.poolLength
	}
	race, ok := c.store.GetRace(event)
	if !ok || race.Distance == nil || race.Distance.Length <= 0 {
		return 50
	}
	laps := race.Distance.Length / 50
	if laps <= 0 {
		return 50
	}
	return race.Distance.Length / laps
}

// logFrame logs the raw timing frame for debugging.
func (c *Connector) logFrame(line string) {
	var hexes strings.Builder
	for i := 0; i < len(line); i++ {
		hexes.WriteString(fmt.Sprintf("%02X ", line[i]))
	}

	hexData := hexes.String()
	c.logger.Debug().Str("hex", hexData).Msg("raw frame")

	now := time.Now()
	entry := logEntry{
		timestamp: now,
		msgType:   string(line[offsetMsgType]),
		msgKind:   string(line[offsetMsgKind]),
		timeType:  string(line[offsetTimeType]),
		usedLanes: fmt.Sprintf("%02X%02X", line[offsetUsedLanes], line[offsetUsedLanes+1]),
		lapCount:  strings.TrimSpace(line[offsetLapCount:offsetLapCountEnd]),
		event:     strings.TrimSpace(line[offsetEvent:offsetEventEnd]),
		heat:      strings.TrimSpace(line[offsetHeat:offsetHeatEnd]),
		rank:      strings.TrimSpace(line[offsetRank:offsetRankEnd]),
		lane:      strings.TrimSpace(line[offsetLane:offsetLaneEnd]),
		lap:       strings.TrimSpace(line[offsetLap:offsetLapEnd]),
		time:      line[offsetTime:offsetTimeEnd],
		hexData:   hexData,
	}

	if line[offsetMsgType] == msgTypeRunning && line[offsetMsgKind] == msgKindStart {
		omegaMS := convertTimeToMS(line[offsetTime:offsetTimeEnd])
		localMS := localTimeOfDayMS(now)
		if omegaMS > 0 {
			entry.omegaTimeMS = strconv.Itoa(omegaMS)
			entry.localDayMS = strconv.Itoa(localMS)
			currentOffset := time.Duration(localMS-omegaMS) * time.Millisecond
			entry.clockOffset = c.clockOffset.String()
			if c.clockSynced {
				delay := currentOffset - c.clockOffset
				if delay < 0 {
					delay = 0
				}
				entry.delay = delay.String()
			}
		}
	}

	select {
	case c.logChannel <- entry:
	default:
		// Log buffer full, discard to prevent blocking the connector
	}
}

// loggerWorker logs the raw timing frame as hex values to a csv file in the race data dir.
func (c *Connector) loggerWorker(logChan chan logEntry) {
	file, writer, err := c.createLogFile()
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to create connector debug log file")
		return
	}
	defer func() {
		writer.Flush()
		_ = file.Close()
	}()

	_ = writer.Write([]string{
		"Timestamp", "MsgType", "MsgKind", "TimeType", "UsedLanes",
		"LapCount", "Event", "Heat", "Rank", "Lane", "Lap", "Time",
		"OmegaTimeMS", "LocalDayMS", "ClockOffset", "Delay", "Hex",
	})
	writer.Flush()

	for entry := range logChan {
		timestamp := entry.timestamp.Format("2006-01-02 15-04-05.000")

		_ = writer.Write([]string{
			timestamp,
			entry.msgType,
			entry.msgKind,
			entry.timeType,
			entry.usedLanes,
			entry.lapCount,
			entry.event,
			entry.heat,
			entry.rank,
			entry.lane,
			entry.lap,
			entry.time,
			entry.omegaTimeMS,
			entry.localDayMS,
			entry.clockOffset,
			entry.delay,
			entry.hexData,
		})
		writer.Flush()
	}
}

// createLogFile creates a CSV file for logging raw timing frames.
func (c *Connector) createLogFile() (*os.File, *csv.Writer, error) {
	logsDir := filepath.Join(c.raceDataDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, nil, err
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := filepath.Join(logsDir, fmt.Sprintf("connector_data_%s.csv", timestamp))

	csvFile, err := os.Create(filename)
	if err != nil {
		return nil, nil, err
	}

	c.logger.Debug().Str("file", filename).Msg("created connector debug file")

	csvWriter := csv.NewWriter(csvFile)
	csvWriter.UseCRLF = true

	return csvFile, csvWriter, nil
}

// localTimeOfDayMS returns the number of milliseconds elapsed since midnight for the given time.
func localTimeOfDayMS(t time.Time) int {
	h, m, s := t.Clock()
	ms := t.Nanosecond() / 1_000_000
	return (h*3600+m*60+s)*1000 + ms
}

// convertTimeToMS converts a time string in the format HH:MM:SS.dd to milliseconds.
func convertTimeToMS(t string) int {
	if len(t) != timeLength {
		return 0
	}

	hours, _ := strconv.Atoi(strings.TrimSpace(t[timeOffsetHours:timeOffsetHoursE]))
	minutes, _ := strconv.Atoi(strings.TrimSpace(t[timeOffsetMin:timeOffsetMinE]))
	seconds, _ := strconv.Atoi(strings.TrimSpace(t[timeOffsetSec:timeOffsetSecE]))
	centiSeconds, _ := strconv.Atoi(strings.TrimSpace(t[timeOffsetCenti:timeOffsetCentiE]))

	return (hours*3600+minutes*60+seconds)*1000 + centiSeconds*10
}
