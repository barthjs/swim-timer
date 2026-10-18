package service

import (
	"encoding/json/v2"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/store"
)

// overviewFiles describes a meet over two days. The file lists the events in the order of the program:
// event 3 on the first day, then on the second day event 2, the final 5 without a scheduled time,
// and event 9, which references a distance, style, and category that do not exist.
// Competitor 2 sorts before competitor 1 by last name, and competitor 2 withdrew from event 5.
var overviewFiles = map[string]string{
	"LSTCAT.TXT":   "Category;AbrevCat\nOpen;\"O\"\n",
	"LSTLONG.TXT":  "idLength;Longueur;Mlongueur;Relais\n1;\"100 m\";100;1\n",
	"LSTSTYLE.TXT": "idStyle;Style;StyleAbrév\n3;\"Backstroke\";\"BA\"\n",
	"LSTRACE.TXT": "event;round;nbHeat;idLen;idStyle;abCat;date;time\n" +
		"3;0;2;1;3;\"O\";\"26/05/01\";\"16:30\"\n" +
		"2;0;1;1;3;\"O\";\"26/05/02\";\"09:00\"\n" +
		"5;3;1;1;3;\"O\";\"26/05/02\";\"00:00\"\n" +
		"9;0;1;7;7;\"X\";\"26/05/02\";\"08:00\"\n",
	"LSTCONC.TXT": "id;bib;lastname;firstname;birthyear;abNat;abCat\n" +
		"1;;\"Young\";\"Zora\";\"2002\";\"SC B\";\"O\";\n" +
		"2;;\"Adams\";\"Anna\";\"2001\";\"SC A\";\"O\";\n",
	"LSTSTART.TXT": "event;round;heat;lane;relais;idBib\n" +
		"3;0;1;4;0;1;\n" +
		"3;0;0;5;0;2;\n" +
		"2;0;0;3;0;1;\n" +
		"5;3;-1;4;0;2;\n",
}

func TestMeetService_ListFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns the files", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, store.NewMeetStore(newTestStorage(t)), nil)
		events := newEventsFile(apitest.Faker(t).IntRange(1, 999), 1)
		require.NoError(t, svc.SaveFile(ctx, domain.MeetFileEvents, []byte(events)))

		names, err := svc.ListFiles(ctx)

		require.NoError(t, err)
		assert.Equal(t, "LSTRACE.TXT", names[domain.MeetFileEvents])
	})

	t.Run("returns no files for an empty storage", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, store.NewMeetStore(newTestStorage(t)), nil)

		names, err := svc.ListFiles(t.Context())

		require.NoError(t, err)
		assert.Empty(t, names)
	})
}

func TestMeetService_ListEvents(t *testing.T) {
	t.Parallel()

	t.Run("returns the events in program order", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		list := svc.ListEvents()

		numbers := make([]int, 0, len(list.Events))
		for _, event := range list.Events {
			numbers = append(numbers, event.Number)
		}
		assert.Equal(t, []int{3, 2, 5, 9}, numbers, "events are ordered by date, then by their position in the file")
		assert.False(t, list.LoadedAt.IsZero())
	})

	t.Run("returns an empty list without meet files", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, nil), nil)

		list := svc.ListEvents()

		assert.NotNil(t, list.Events)
		assert.Empty(t, list.Events)
	})
}

func TestMeetService_ListCompetitors(t *testing.T) {
	t.Parallel()

	t.Run("returns the competitors ordered by name", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		list := svc.ListCompetitors()

		require.Len(t, list.Competitors, 2)
		assert.Equal(t, "Adams", list.Competitors[0].LastName)
		assert.Equal(t, "Young", list.Competitors[1].LastName)
		assert.False(t, list.LoadedAt.IsZero())
	})

	t.Run("returns an empty list without meet files", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, nil), nil)

		list := svc.ListCompetitors()

		assert.NotNil(t, list.Competitors)
		assert.Empty(t, list.Competitors)
	})
}

func TestMeetService_ListStartEntries(t *testing.T) {
	t.Parallel()

	t.Run("returns the start entries in the order of the heats", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		list := svc.ListStartEntries()

		assert.Equal(t, []MeetStartEntry{
			{
				Event:        2,
				Heat:         1,
				Lane:         3,
				CompetitorID: 1,
			},
			{
				Event:        3,
				Heat:         1,
				Lane:         5,
				CompetitorID: 2,
			},
			{
				Event:        3,
				Heat:         2,
				Lane:         4,
				CompetitorID: 1,
			},
		}, list.StartEntries, "heats are counted from 1, and the withdrawn entry is left out")
		assert.False(t, list.LoadedAt.IsZero())
	})

	t.Run("returns an empty list without meet files", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, nil), nil)

		list := svc.ListStartEntries()

		assert.NotNil(t, list.StartEntries)
		assert.Empty(t, list.StartEntries)
	})
}

func TestMeetService_GetOverview(t *testing.T) {
	t.Parallel()

	t.Run("returns the overview in program order", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		overview := svc.GetOverview()

		numbers := make([]int, 0, len(overview.Events))
		for _, event := range overview.Events {
			numbers = append(numbers, event.Number)
		}
		assert.Equal(t, []int{3, 2, 5, 9}, numbers, "events are ordered by date, then by their position in the file")
		assert.Equal(t, []MeetStartEntry{
			{
				Event:        2,
				Heat:         1,
				Lane:         3,
				CompetitorID: 1,
			},
			{
				Event:        3,
				Heat:         1,
				Lane:         5,
				CompetitorID: 2,
			},
			{
				Event:        3,
				Heat:         2,
				Lane:         4,
				CompetitorID: 1,
			},
		}, overview.StartEntries, "heats are counted from 1, and the withdrawn entry is left out")
		require.Len(t, overview.Competitors, 2)
		assert.Equal(t, "Adams", overview.Competitors[0].LastName)
		assert.False(t, overview.LoadedAt.IsZero())
	})

	t.Run("returns empty lists without meet files", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, nil), nil)

		overview := svc.GetOverview()

		assert.NotNil(t, overview.Events)
		assert.NotNil(t, overview.StartEntries)
		assert.NotNil(t, overview.Competitors)
		assert.Empty(t, overview.Events)
	})

	t.Run("leaves out missing references", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		overview := svc.GetOverview()

		for _, event := range overview.Events {
			if event.Number != 9 {
				continue
			}
			assert.Nil(t, event.Distance)
			assert.Nil(t, event.Style)
			assert.Nil(t, event.Category)
			assert.Empty(t, event.DisplayName())
			return
		}
		t.Fatal("event 9 is missing")
	})
}

func TestMeetService_GetEvent(t *testing.T) {
	t.Parallel()

	t.Run("returns the event", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		event, err := svc.GetEvent(5)

		require.NoError(t, err)
		assert.Equal(t, 5, event.Event.Number)
		assert.Equal(t, 3, event.Event.PhaseID)
		assert.False(t, event.LoadedAt.IsZero())
	})

	t.Run("returns not found for an unknown event", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		_, err := svc.GetEvent(4)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

// heatFiles describes event 4 in phase 3 with one heat. Its start list holds two relay positions in lane 2,
// an entry of another phase, and a withdrawn entry, and no competitor file exists.
var heatFiles = map[string]string{
	"LSTRACE.TXT": "event;round;nbHeat;idLen;idStyle;abCat;date;time\n" +
		"4;3;1;1;3;\"O\";\"26/05/01\";\"10:00\"\n",
	"LSTSTART.TXT": "event;round;heat;lane;relais;idBib\n" +
		"4;3;0;6;0;1;\n" +
		"4;0;0;1;0;2;\n" +
		"4;3;0;2;2;3;\n" +
		"4;3;0;2;1;4;\n" +
		"4;3;-1;3;0;5;\n",
}

func TestMeetService_GetHeat(t *testing.T) {
	t.Parallel()

	t.Run("returns the event and the start entries of the heat", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		heat, err := svc.GetHeat(3, 2)

		require.NoError(t, err)
		assert.Equal(t, 3, heat.Event.Number)
		require.Len(t, heat.StartEntries, 1)
		assert.Equal(t, 4, heat.StartEntries[0].Lane)
		require.NotNil(t, heat.StartEntries[0].Competitor)
		assert.Equal(t, "Young", heat.StartEntries[0].Competitor.LastName)
		assert.False(t, heat.LoadedAt.IsZero())
	})

	t.Run("orders the start entries of the phase by lane and relay position", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, heatFiles), nil)

		heat, err := svc.GetHeat(4, 1)

		require.NoError(t, err)
		competitorIDs := make([]int, 0, len(heat.StartEntries))
		for _, entry := range heat.StartEntries {
			competitorIDs = append(competitorIDs, entry.CompetitorID)
			assert.Nil(t, entry.Competitor, "the competitor does not exist")
		}
		assert.Equal(t, []int{4, 3, 1}, competitorIDs, "the other phase and the withdrawn entry are left out")
	})

	t.Run("returns an empty heat without start entries", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		heat, err := svc.GetHeat(9, 1)

		require.NoError(t, err)
		assert.NotNil(t, heat.StartEntries)
		assert.Empty(t, heat.StartEntries)
	})

	t.Run("returns not found for an unknown event", func(t *testing.T) {
		t.Parallel()

		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

		_, err := svc.GetHeat(4, 1)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("returns not found for a heat outside the event", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			heat int
		}{
			{
				name: "before the first heat",
				heat: 0,
			},
			{
				name: "after the last heat",
				heat: 3,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, newTestMeetStore(t, overviewFiles), nil)

				_, err := svc.GetHeat(3, tt.heat)

				assert.ErrorIs(t, err, apperr.ErrNotFound)
			})
		}
	})
}

func TestMeetService_SaveFile(t *testing.T) {
	t.Parallel()

	t.Run("keeps the meet until the next reload", func(t *testing.T) {
		t.Parallel()

		meetStore := store.NewMeetStore(newTestStorage(t))
		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, meetStore, nil)
		number := apitest.Faker(t).IntRange(1, 999)
		events := newEventsFile(number, 1)

		err := svc.SaveFile(t.Context(), domain.MeetFileEvents, []byte(events))

		require.NoError(t, err)
		_, ok := meetStore.GetEvent(number)
		assert.False(t, ok, "an upload is only read on the next reload")
	})
}

func TestMeetService_Reload(t *testing.T) {
	t.Parallel()

	t.Run("reads the stored files", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		meetStore := store.NewMeetStore(newTestStorage(t))
		broadcaster := &fakeBroadcaster{}
		svc := NewMeetService(zerolog.Nop(), broadcaster, meetStore, nil)
		faker := apitest.Faker(t)
		number := faker.IntRange(1, 999)
		heatCount := faker.IntRange(1, 20)
		events := newEventsFile(number, heatCount)
		require.NoError(t, svc.SaveFile(ctx, domain.MeetFileEvents, []byte(events)))

		err := svc.Reload(ctx)

		require.NoError(t, err)
		assert.Equal(t, 1, broadcaster.count(), "a successful reload notifies the open pages")
		got, ok := meetStore.GetEvent(number)
		require.True(t, ok)
		assert.Equal(t, heatCount, got.HeatCount)
	})

	t.Run("notifies the open pages", func(t *testing.T) {
		t.Parallel()

		broadcaster := &fakeBroadcaster{}
		svc := NewMeetService(zerolog.Nop(), broadcaster, store.NewMeetStore(newTestStorage(t)), nil)

		err := svc.Reload(t.Context())

		require.NoError(t, err)
		require.Equal(t, 1, broadcaster.count())
		var message map[string]any
		require.NoError(t, json.Unmarshal(broadcaster.messages[0], &message))
		assert.Equal(t, "meet", message["type"])
		assert.NotEmpty(t, message["serverTime"])
		assert.NotContains(t, message, "data")
	})

	t.Run("does not notify after a failed reload", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		broadcaster := &fakeBroadcaster{}
		svc := NewMeetService(zerolog.Nop(), broadcaster, store.NewMeetStore(newTestStorage(t)), nil)
		faker := apitest.Faker(t)
		broken := "event;round;nbHeat\n" + faker.LetterN(4) + ";" + faker.LetterN(4) + ";" + faker.LetterN(4) + "\n"
		require.NoError(t, svc.SaveFile(ctx, domain.MeetFileEvents, []byte(broken)))

		err := svc.Reload(ctx)

		require.Error(t, err)
		assert.Zero(t, broadcaster.count())
	})

	t.Run("runs the reload callback after a successful reload", func(t *testing.T) {
		t.Parallel()

		calls := 0
		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, store.NewMeetStore(newTestStorage(t)), func() { calls++ })

		err := svc.Reload(t.Context())

		require.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("skips the reload callback after a failed reload", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		calls := 0
		svc := NewMeetService(zerolog.Nop(), &fakeBroadcaster{}, store.NewMeetStore(newTestStorage(t)), func() { calls++ })
		require.NoError(t, svc.SaveFile(ctx, domain.MeetFileEvents, []byte("event;round;nbHeat\nx;y;z\n")))

		err := svc.Reload(ctx)

		require.Error(t, err)
		assert.Zero(t, calls, "the meet files in use did not change")
	})
}

// newEventsFile returns an events file with a single event of the given number and heat count.
func newEventsFile(number, heatCount int) string {
	return fmt.Sprintf("event;round;nbHeat;idLen;idStyle;abCat;date;time\n%d;0;%d;0;0;\"O\";\"26/05/01\";\"10:00\"\n", number, heatCount)
}

func TestCompareCompetitors(t *testing.T) {
	t.Parallel()

	t.Run("sorts accents with their letter", func(t *testing.T) {
		t.Parallel()

		names := collate.New(language.Und)
		accented := &domain.Competitor{
			LastName: "Ébert",
		}
		plain := &domain.Competitor{
			LastName: "Zola",
		}

		order := compareCompetitors(names, accented, plain)

		assert.Negative(t, order, "É sorts next to E, not after Z")
	})
}
