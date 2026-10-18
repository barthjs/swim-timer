package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"github.com/barthjs/swim-timer/internal/api"
	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/config"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/omega"
	"github.com/barthjs/swim-timer/internal/recording"
)

// newAuthLoginRequest returns a login with a random password, which never matches the admin password.
func newAuthLoginRequest(f *gofakeit.Faker) api.AuthLoginRequest {
	return api.AuthLoginRequest{
		Password: f.Password(true, true, true, false, false, 20),
	}
}

// newS3StorageSettings returns storage settings of the s3 driver with every field the driver needs.
func newS3StorageSettings(f *gofakeit.Faker) api.StorageSettings {
	return api.StorageSettings{
		Driver: string(media.DriverS3),
		S3: api.S3Settings{
			Endpoint:  f.URL(),
			Bucket:    f.LetterN(10),
			AccessKey: f.LetterN(16),
			SecretKey: f.LetterN(32),
		},
	}
}

// newConfigRequest returns a valid save request for cfg with random values.
func newConfigRequest(f *gofakeit.Faker, cfg config.Config) api.ConfigSaveRequest {
	return api.ConfigSaveRequest{
		App: api.AppSettings{
			Name:     f.AppName(),
			Env:      string(cfg.App.Env),
			Secret:   f.Password(true, true, true, false, false, 16),
			Language: string(cfg.App.Language),
		},
		Log: api.LogSettings{
			Level:   cfg.Log.RawLevel,
			Channel: string(cfg.Log.Channel),
			Format:  string(cfg.Log.Format),
		},
		Server: api.ServerSettings{
			Host:               cfg.Server.Host,
			Port:               f.IntRange(1024, 65535),
			CORSAllowedOrigins: cfg.Server.CORSAllowedOrigins,
		},
		Storage: api.StorageSettings{
			Driver: string(cfg.Storage.Driver),
			Path:   cfg.Storage.Path,
		},
		Theme: api.ThemeSettings{
			Primary: f.HexColor(),
		},
		Timing: api.TimingSettings{
			SerialPort: f.LetterN(10),
			BaudRate:   cfg.Timing.BaudRate,
			PoolLength: f.IntRange(0, 100),
		},
	}
}

// meetFixture is the content of the export files of a small meet with one event,
// whose heats have a competitor in each of their eight lanes.
type meetFixture struct {
	category    domain.Category
	distance    domain.Distance
	style       domain.Style
	phase       domain.Phase
	event       domain.Event
	competitors []domain.Competitor
	entries     []domain.StartEntry
}

const meetLaneCount = 8

// newMeetFixture returns a meet with heatCount heats.
func newMeetFixture(f *gofakeit.Faker, heatCount int) meetFixture {
	// At least two laps in any pool, so a split comes before the finish
	length := 100 * f.IntRange(1, 8)
	m := meetFixture{
		category: domain.Category{
			Name:         f.Noun(),
			Abbreviation: strings.ToUpper(f.LetterN(2)),
		},
		distance: domain.Distance{
			ID:     f.IntRange(1, 99),
			Name:   fmt.Sprintf("%d m", length),
			Length: length,
			Relay:  1,
		},
		style: domain.Style{
			ID:           f.IntRange(0, 99),
			Name:         f.Noun(),
			Abbreviation: strings.ToUpper(f.LetterN(2)),
		},
		phase: domain.Phase{
			ID:           f.IntRange(0, 5),
			Name:         f.Noun(),
			Abbreviation: strings.ToUpper(f.LetterN(3)),
		},
	}
	m.event = domain.Event{
		Number:     f.IntRange(1, 999),
		PhaseID:    m.phase.ID,
		HeatCount:  heatCount,
		DistanceID: m.distance.ID,
		StyleID:    m.style.ID,
		AbCat:      m.category.Abbreviation,
		Date:       f.Date().Format("06/01/02"),
		Time:       fmt.Sprintf("%02d:%02d", f.IntRange(7, 20), f.IntRange(0, 59)),
	}

	id := f.IntRange(1, 1000)
	for heat := range heatCount {
		for lane := 1; lane <= meetLaneCount; lane++ {
			competitor := domain.Competitor{
				ID:        id,
				BIB:       id,
				LastName:  f.LastName(),
				FirstName: f.FirstName(),
				BirthYear: strconv.Itoa(f.IntRange(1990, 2018)),
				AbNat:     f.Company(),
				AbCat:     m.category.Abbreviation,
			}
			m.competitors = append(m.competitors, competitor)
			m.entries = append(m.entries, domain.StartEntry{
				Event:        m.event.Number,
				PhaseID:      m.event.PhaseID,
				Heat:         heat,
				Lane:         lane,
				CompetitorID: id,
			})
			id++
		}
	}
	return m
}

// competitor returns the competitor in a lane of a heat counted from 1.
func (m meetFixture) competitor(heat, lane int) domain.Competitor {
	for _, entry := range m.entries {
		if entry.Heat == heat-1 && entry.Lane == lane {
			for _, competitor := range m.competitors {
				if competitor.ID == entry.CompetitorID {
					return competitor
				}
			}
		}
	}
	panic(fmt.Sprintf("no competitor in heat %d lane %d", heat, lane))
}

// displayName is the event name the application composes from distance, style, and category.
func (m meetFixture) displayName() string {
	return m.distance.Name + " " + m.style.Name + " " + m.category.Name
}

// loadMeet uploads the export files of a meet and reloads the meet data, as an admin does before a competition.
func (h *harness) loadMeet(m meetFixture) {
	h.t.Helper()

	admin := h.asAdmin()
	admin.POST("/api/meet/files", apitest.Multipart{
		Files: m.files(h.t),
	}).AssertOK()
	admin.POST("/api/meet/reload", nil).AssertNoContent()
}

// files returns every export file as an upload, Windows-1252 encoded as the timing software writes them.
func (m meetFixture) files(t *testing.T) []apitest.File {
	t.Helper()

	files := make([]apitest.File, 0, len(domain.MeetFiles()))
	for _, file := range domain.MeetFiles() {
		files = append(files, apitest.File{
			Field:   "files",
			Name:    string(file) + ".TXT",
			Content: windows1252(t, m.csv()[file]),
		})
	}
	return files
}

// windows1252 encodes text as the timing software writes its export files.
func windows1252(t *testing.T, text string) []byte {
	t.Helper()

	encoded, err := charmap.Windows1252.NewEncoder().String(text)
	require.NoError(t, err)
	return []byte(encoded)
}

// csv returns the rows of every export file in the layout and quoting of the timing software.
func (m meetFixture) csv() map[domain.MeetFile]string {
	var competitors, entries strings.Builder
	_, _ = competitors.WriteString("id;bib;lastname;firstname;birthyear;abNat;abCat\r\n")
	for _, c := range m.competitors {
		_, _ = fmt.Fprintf(
			&competitors,
			"%d;\"%d\";\"%s\";\"%s\";\"%s\";\"%s\";\"%s\";\r\n",
			c.ID,
			c.BIB,
			c.LastName,
			c.FirstName,
			c.BirthYear,
			c.AbNat,
			c.AbCat,
		)
	}
	_, _ = entries.WriteString("event;round;heat;lane;relais;idBib\r\n")
	for _, e := range m.entries {
		_, _ = fmt.Fprintf(&entries, "%d; %d; %d; %d; 0; %d; \r\n", e.Event, e.PhaseID, e.Heat, e.Lane, e.CompetitorID)
	}

	return map[domain.MeetFile]string{
		domain.MeetFileCategories: fmt.Sprintf("\"Category\";\"AbrevCat\"\r\n\"%s\";\"%s\"\r\n", m.category.Name, m.category.Abbreviation),
		domain.MeetFileDistances: fmt.Sprintf(
			"\"idLength\";\"Longueur\";\"Mlongueur\";\"Relais\"\r\n %d;\"%s\"; %d; %d\r\n",
			m.distance.ID,
			m.distance.Name,
			m.distance.Length,
			m.distance.Relay,
		),
		domain.MeetFileStyles:      fmt.Sprintf("idStyle;Style;StyleAbrév\r\n%d; \"%s\"; \"%s\"\r\n", m.style.ID, m.style.Name, m.style.Abbreviation),
		domain.MeetFileCompetitors: competitors.String(),
		domain.MeetFileEvents: fmt.Sprintf(
			"event;round;nbHeat;idLen;idStyle;abCat;date;time\r\n%d;%d; %d; %d; %d; \"%s\"; \"%s\";\"%s\"; \r\n",
			m.event.Number,
			m.event.PhaseID,
			m.event.HeatCount,
			m.event.DistanceID,
			m.event.StyleID,
			m.event.AbCat,
			m.event.Date,
			m.event.Time,
		),
		domain.MeetFilePhases: fmt.Sprintf(
			"idRound;TITLE;RoundAbrev;roundtext;sequence\r\n%d;\"%s\";\"%s\";\"Heat\";\"1\"\r\n",
			m.phase.ID,
			m.phase.Name,
			m.phase.Abbreviation,
		),
		domain.MeetFileStartEntries: entries.String(),
	}
}

// newPresetSaveRequest returns a valid preset with random lanes.
func newPresetSaveRequest(f *gofakeit.Faker, opts ...func(*api.PresetSaveRequest)) api.PresetSaveRequest {
	comment := f.LoremIpsumSentence(6)
	laneCount := f.IntRange(2, 10)
	left := f.Float64Range(0, 0.2)
	right := left + f.Float64Range(0.3, 0.7)
	height := f.Float64Range(0.02, 0.07)

	lanes := make(map[string]api.PresetLaneRequest, laneCount)
	for lane := 1; lane <= laneCount; lane++ {
		top := 0.01 + float64(lane-1)*(height+0.005)
		lanes[strconv.Itoa(lane)] = api.PresetLaneRequest{
			TL: api.PresetPointRequest{
				X: left,
				Y: top,
			},
			TR: api.PresetPointRequest{
				X: right,
				Y: top,
			},
			BR: api.PresetPointRequest{
				X: right,
				Y: top + height,
			},
			BL: api.PresetPointRequest{
				X: left,
				Y: top + height,
			},
			Locked: f.Bool(),
		}
	}

	preset := api.PresetSaveRequest{
		Name:         presetName(f),
		Comment:      &comment,
		FrameWidth:   f.IntRange(640, 3840),
		FrameHeight:  f.IntRange(360, 2160),
		TextSize:     f.IntRange(10, 100),
		LaneGap:      f.IntRange(0, 100),
		ReverseLanes: f.Bool(),
		TextAnchor: domain.TextAnchor(f.RandomString([]string{
			string(domain.TextAnchorLeft),
			string(domain.TextAnchorRight),
		})),
		Lanes: lanes,
	}
	for _, opt := range opts {
		opt(&preset)
	}
	return preset
}

// presetName returns a name that addresses a preset file, so it holds no path separator.
func presetName(f *gofakeit.Faker) string {
	return f.Noun() + "-" + f.LetterN(6)
}

// storeRecording uploads a recording with a random name and returns it.
func (h *harness) storeRecording() api.RecordingResponse {
	h.t.Helper()

	return h.asAdmin().POST("/api/recordings", newRecordingUpload(h.t, h.Faker)).AssertCreated().Decode[api.RecordingResponse]()
}

// storeRecordingNamed uploads a recording with the given name and returns it.
func (h *harness) storeRecordingNamed(name string) api.RecordingResponse {
	h.t.Helper()

	upload := newRecordingUpload(h.t, h.Faker)
	upload.Files[0].Name = name
	return h.asAdmin().POST("/api/recordings", upload).AssertCreated().Decode[api.RecordingResponse]()
}

// currentRecording waits until the timing service has opened its recording and returns it.
func (h *harness) currentRecording() api.RecordingResponse {
	h.t.Helper()

	var current api.RecordingResponse
	require.Eventually(h.t, func() bool {
		for _, item := range h.asAdmin().GET("/api/recordings").AssertOK().Decode[[]api.RecordingResponse]() {
			if item.Current {
				current = item
				return true
			}
		}
		return false
	}, 5*time.Second, 10*time.Millisecond)
	return current
}

// newRecordingUpload returns an upload of a recording with one session and one frame under a random name.
func newRecordingUpload(t *testing.T, f *gofakeit.Faker) apitest.Multipart {
	t.Helper()

	at := time.Now()
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	require.NoError(t, encoder.Encode(recording.Entry{
		Type: recording.TypeSession,
		At:   at,
		Session: &recording.Session{
			Format:  recording.FormatVersion,
			Version: f.AppVersion(),
			Source:  recording.SourceDevice,
		},
	}))
	require.NoError(t, encoder.Encode(recording.Entry{
		Type: recording.TypeData,
		At:   at.Add(time.Second),
		Data: omega.Encode(omega.FrameData{
			Kind:     omega.KindLoad,
			LapCount: 2,
			Event:    f.IntRange(1, 999),
			Heat:     f.IntRange(1, 9),
		}),
	}))

	return apitest.Multipart{
		Files: []apitest.File{
			{
				Field:   "file",
				Name:    f.LetterN(10) + ".jsonl",
				Content: content.Bytes(),
			},
		},
	}
}
