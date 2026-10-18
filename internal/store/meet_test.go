package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"github.com/barthjs/swim-timer/internal/domain"
)

const (
	testStyles       = "idStyle;Style;StyleAbrév\r\n1;\"Breaststroke\";\"BR\"\r\n"
	testCompetitors  = "id;bib;lastname;firstname;birthyear;abNat;abCat\r\n7;;\"Brontë\";\"Chloé\";\"2010\";\"SC South\";\"M\";\r\n"
	testEvents       = "event;round;nbHeat;idLen;idStyle;abCat;date;time\r\n1;1;1;1;1;\"W\";\"26/05/01\";\"16:00\";\r\n"
	testStartEntries = "event;round;heat;lane;relais;idBib\r\n1;1;0;1;0;1;\r\n1;1;0;2;0;2;\r\n"
)

func TestMeetStore_ListFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns the file with every extension", func(t *testing.T) {
		t.Parallel()

		for _, ext := range meetExtensions {
			t.Run(ext, func(t *testing.T) {
				t.Parallel()

				storage, root := newTestStorageWithRoot(t)
				writeMeetFile(t, root, "LSTSTYLE"+ext, testStyles)

				names, err := NewMeetStore(storage).ListFiles(t.Context())

				require.NoError(t, err)
				assert.Equal(t, "LSTSTYLE"+ext, names[domain.MeetFileStyles])
			})
		}
	})

	t.Run("prefers the uppercase extension", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		// The uppercase extension of the export wins over a lowercase copy
		writeMeetFile(t, root, "LSTSTYLE.txt", "idStyle;Style;StyleAbrév\r\n1;\"Backstroke\";\"BA\"\r\n")
		writeMeetFile(t, root, "LSTSTYLE.TXT", testStyles)

		names, err := NewMeetStore(storage).ListFiles(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "LSTSTYLE.TXT", names[domain.MeetFileStyles])
		assert.NotContains(t, names, domain.MeetFileCompetitors, "a missing file has no entry")
	})
}

func TestMeetStore_ListFileContents(t *testing.T) {
	t.Parallel()

	t.Run("returns the loaded files as they were read", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		st := NewMeetStore(storage)
		writeMeetFile(t, root, "LSTSTART.TXT", "event;round;heat;lane;relais;idBib\r\n101;0;0;4;0;46;\r\n")
		writeMeetFile(t, root, "LSTCONC.TXT", "id;bib;lastname;firstname;birthyear;abNat;abCat\r\n46;;\"Doré\";\"Zoë\";\"2016\";\"SC Riverside\";\"W\";\r\n")
		require.NoError(t, st.LoadAll(t.Context()))

		got := st.ListFileContents()

		require.Len(t, got, 2)
		assert.Equal(t, domain.MeetFileCompetitors, got[0].File, "in the order of the meet files")
		assert.Equal(t, domain.MeetFileStartEntries, got[1].File)
		stored, err := os.ReadFile(filepath.Join(root, meetFolder, "LSTCONC.TXT"))
		require.NoError(t, err)
		assert.Equal(t, stored, got[0].Content, "the content in its original encoding")
	})

	t.Run("keeps the files of the last successful load", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		st := NewMeetStore(storage)
		writeMeetFile(t, root, "LSTSTART.TXT", "event;round;heat;lane;relais;idBib\r\n101;0;0;4;0;46;\r\n")
		require.NoError(t, st.LoadAll(t.Context()))
		before := st.ListFileContents()
		writeMeetFile(t, root, "LSTSTART.TXT", "event;round;heat;lane;relais;idBib\r\n101;x;0;4;0;46;\r\n")
		require.Error(t, st.LoadAll(t.Context()))

		got := st.ListFileContents()

		assert.Equal(t, before, got)
	})

	t.Run("returns nothing before the first load", func(t *testing.T) {
		t.Parallel()

		st := NewMeetStore(newTestStorage(t))

		got := st.ListFileContents()

		assert.Empty(t, got)
	})
}

func TestMeetStore_GetHeat(t *testing.T) {
	t.Parallel()

	t.Run("returns the heat", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTROUND.TXT", "idRound;TITLE;RoundAbrev;roundtext;sequence\r\n1;\"Heats\";\"HEA\";\"Heat\";\"1\"\r\n")
		writeMeetFile(t, root, "LSTRACE.TXT", testEvents)
		writeMeetFile(t, root, "LSTSTART.TXT", testStartEntries)
		st := NewMeetStore(storage)
		require.NoError(t, st.LoadAll(t.Context()))

		heat, ok := st.GetHeat(1, 0)

		require.True(t, ok)
		assert.Equal(t, "Heats", heat.Event.PhaseName(), "the phase is linked from LSTROUND")
		assert.Len(t, heat.StartEntries, 2, "the entries of the phase of the event")
		assert.Equal(t, 1, heat.FirstLane)
		assert.False(t, heat.LoadedAt.IsZero())
	})

	firstLanes := []struct {
		name   string
		starts string
		want   int
	}{
		{
			name:   "returns the first lane of a pool numbered from 1",
			starts: "event;round;heat;lane;relais;idBib\r\n1;1;0;1;0;1;\r\n1;1;0;10;0;2;\r\n",
			want:   1,
		},
		{
			name:   "returns the first lane of a pool numbered from 0",
			starts: "event;round;heat;lane;relais;idBib\r\n1;1;0;0;0;1;\r\n1;1;0;9;0;2;\r\n",
			want:   0,
		},
		{
			name:   "ignores a withdrawn entry in lane 0 for the first lane",
			starts: "event;round;heat;lane;relais;idBib\r\n1;1;-1;0;0;1;\r\n1;1;0;10;0;2;\r\n",
			want:   1,
		},
	}

	for _, tt := range firstLanes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage, root := newTestStorageWithRoot(t)
			writeMeetFile(t, root, "LSTSTART.TXT", tt.starts)
			st := NewMeetStore(storage)
			require.NoError(t, st.LoadAll(t.Context()))

			heat, _ := st.GetHeat(1, 0)

			assert.Equal(t, tt.want, heat.FirstLane)
		})
	}

	t.Run("returns not found for an unknown event", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTRACE.TXT", testEvents)
		writeMeetFile(t, root, "LSTSTART.TXT", testStartEntries)
		st := NewMeetStore(storage)
		require.NoError(t, st.LoadAll(t.Context()))

		_, ok := st.GetHeat(2, 0)

		assert.False(t, ok)
	})
}

func TestMeetStore_SaveFile(t *testing.T) {
	t.Parallel()

	t.Run("stores the file", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		st := NewMeetStore(newTestStorage(t))
		competitors, err := charmap.Windows1252.NewEncoder().String(testCompetitors)
		require.NoError(t, err)
		starts := "event;round;heat;lane;relais;idBib\r\n5;1;0;3;0;7;\r\n"
		require.NoError(t, st.SaveFile(ctx, domain.MeetFileStartEntries, strings.NewReader(starts), int64(len(starts))))

		err = st.SaveFile(ctx, domain.MeetFileCompetitors, strings.NewReader(competitors), -1)

		require.NoError(t, err)
		names, err := st.ListFiles(ctx)
		require.NoError(t, err)
		assert.Equal(t, "LSTCONC.TXT", names[domain.MeetFileCompetitors])
		require.NoError(t, st.LoadAll(ctx))
		entries := startsOf(st, 5, 1, 0)
		require.Len(t, entries, 1)
		require.NotNil(t, entries[0].Competitor)
		assert.Equal(t, 7, entries[0].Competitor.ID)
		assert.Equal(t, "Brontë", entries[0].Competitor.LastName)
	})

	t.Run("replaces a lowercase copy", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTSTYLE.txt", "idStyle;Style;StyleAbrév\r\n1;\"Backstroke\";\"BA\"\r\n")
		st := NewMeetStore(storage)
		encoded, err := charmap.Windows1252.NewEncoder().String(testStyles)
		require.NoError(t, err)

		err = st.SaveFile(ctx, domain.MeetFileStyles, strings.NewReader(encoded), int64(len(encoded)))

		require.NoError(t, err)
		require.NoError(t, st.LoadAll(ctx))
		names, err := st.ListFiles(ctx)
		require.NoError(t, err)
		assert.Equal(t, "LSTSTYLE.TXT", names[domain.MeetFileStyles])
		assert.NoFileExists(t, filepath.Join(root, meetFolder, "LSTSTYLE.txt"), "the copy would hide the upload otherwise")
	})
}

func TestMeetStore_Snapshot(t *testing.T) {
	t.Parallel()

	t.Run("returns the loaded meet", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		st := NewMeetStore(storage)
		writeMeetFile(t, root, "LSTCONC.TXT", "id;bib;lastname;firstname;birthyear;abNat;abCat\r\n46;;\"Doré\";\"Zoë\";\"2016\";\"SC Riverside\";\"W\";\r\n")
		writeMeetFile(t, root, "LSTRACE.TXT", "event;round;nbHeat;idLen;idStyle;abCat;date;time\r\n101;0;1;1;0;\"W\";\"26/05/01\";\"16:00\"\r\n")
		writeMeetFile(t, root, "LSTSTART.TXT", "event;round;heat;lane;relais;idBib\r\n101;0;0;4;0;46;\r\n")
		require.NoError(t, st.LoadAll(t.Context()))

		snapshot := st.Snapshot()

		assert.False(t, snapshot.LoadedAt.IsZero())
		require.Len(t, snapshot.Events, 1)
		require.Len(t, snapshot.StartEntries, 1)
		require.Len(t, snapshot.Competitors, 1)
		assert.Same(t, snapshot.Competitors[0], snapshot.StartEntries[0].Competitor, "the snapshot keeps the resolved references")
	})

	t.Run("returns empty lists before the first load", func(t *testing.T) {
		t.Parallel()

		st := NewMeetStore(newTestStorage(t))

		snapshot := st.Snapshot()

		assert.True(t, snapshot.LoadedAt.IsZero(), "nothing was loaded yet")
		assert.NotNil(t, snapshot.Events)
		assert.NotNil(t, snapshot.StartEntries)
		assert.NotNil(t, snapshot.Competitors)
	})
}

func TestMeetStore_LoadAll(t *testing.T) {
	t.Parallel()

	t.Run("reads the files", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTCAT.TXT", "\"Category\";\"AbrevCat\"\r\n\"Women\";\"W\"\r\n")
		writeMeetFile(t, root, "LSTLONG.TXT", "\"idLength\";\"Longueur\";\"Mlongueur\";\"Relais\"\r\n 1;\"  100 m\"; 100; 1\r\n")
		writeMeetFile(t, root, "LSTSTYLE.TXT", "idStyle;Style;StyleAbrév\r\n0; \"Freestyle    \"; \"FR\"\r\n")
		writeMeetFile(t, root, "LSTCONC.TXT", "id;bib;lastname;firstname;birthyear;abNat;abCat\r\n46;\"146\";\"Doré\";\"Zoë\";\"2016\";\"SC Riverside\";\"W\";\r\n47;\"147\";\"Núñez\";\"Lena\";\"2015\";\"SC North\";\"W\";\r\n")
		writeMeetFile(t, root, "LSTRACE.TXT", "event;round;nbHeat;idLen;idStyle;abCat;date;time\r\n101;1; 2; 1; 0; \"W\"; \"26/05/01\";\"16:00\"; \r\n")
		writeMeetFile(t, root, "LSTSTART.TXT", "event;round;heat;lane;relais;idBib\r\n101; 1; 0; 4; 0; 46; \r\n101; 0; 0; 4; 0; 47; \r\n")
		st := NewMeetStore(storage)

		err := st.LoadAll(t.Context())

		require.NoError(t, err)
		event, ok := st.GetEvent(101)
		require.True(t, ok)
		require.NotNil(t, event.Distance)
		require.NotNil(t, event.Style)
		require.NotNil(t, event.Category)
		assert.Equal(t, "100 m", event.Distance.Name)
		assert.Equal(t, 100, event.Distance.Length)
		assert.Equal(t, "Freestyle", event.Style.Name)
		assert.Equal(t, "FR", event.Style.Abbreviation)
		assert.Equal(t, "Women", event.Category.Name)
		// Only the entry of the event's phase belongs to the heat
		starts := startsOf(st, 101, event.PhaseID, 0)
		require.Len(t, starts, 1)
		// idBib references the id of a competitor, not the start number in the bib column
		require.NotNil(t, starts[0].Competitor)
		assert.Equal(t, "Doré", starts[0].Competitor.LastName, "competitor by id, decoded from Windows-1252")
	})

	t.Run("loads nothing without files", func(t *testing.T) {
		t.Parallel()

		st := NewMeetStore(newTestStorage(t))

		err := st.LoadAll(t.Context())

		require.NoError(t, err, "no file is not an error")
		_, ok := st.GetEvent(1)
		assert.False(t, ok)
		assert.Empty(t, st.ListStartEntries())
	})

	t.Run("keeps the order of the events in the file", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTRACE.TXT", "event;round;nbHeat;idLen;idStyle;abCat;date;time\r\n"+
			"5;0;1;1;1;\"W\";\"26/05/01\";\"16:00\";\r\n"+
			"2;3;1;1;1;\"W\";\"26/05/01\";\"00:00\";\r\n")
		st := NewMeetStore(storage)

		err := st.LoadAll(t.Context())

		require.NoError(t, err)
		first, ok := st.GetEvent(5)
		require.True(t, ok)
		second, ok := st.GetEvent(2)
		require.True(t, ok)
		assert.Equal(t, 0, first.Order)
		assert.Equal(t, 1, second.Order)
	})

	t.Run("keeps both entries of a shared lane", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTSTART.TXT", "event;round;heat;lane;relais;idBib\r\n9;1;0;4;0;1;\r\n9;1;0;4;0;2;\r\n")
		st := NewMeetStore(storage)

		err := st.LoadAll(t.Context())

		require.NoError(t, err)
		entries := startsOf(st, 9, 1, 0)
		require.Len(t, entries, 2)
		assert.Equal(t, 1, entries[0].CompetitorID, "entries keep the file order")
		assert.Equal(t, 2, entries[1].CompetitorID)
	})

	t.Run("replaces the data on a reload", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTSTART.TXT", testStartEntries)
		st := NewMeetStore(storage)
		require.NoError(t, st.LoadAll(ctx))
		require.Len(t, startsOf(st, 1, 1, 0), 2)
		writeMeetFile(t, root, "LSTSTART.TXT", "event;round;heat;lane;relais;idBib\r\n2;1;0;1;0;1;\r\n")

		err := st.LoadAll(ctx)

		require.NoError(t, err)
		assert.Empty(t, startsOf(st, 1, 1, 0), "the old start list is gone")
		assert.Len(t, startsOf(st, 2, 1, 0), 1)
		assert.Len(t, st.ListStartEntries(), 1)
	})

	t.Run("clears the data of a removed file", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTRACE.TXT", testEvents)
		writeMeetFile(t, root, "LSTSTART.TXT", testStartEntries)
		st := NewMeetStore(storage)
		require.NoError(t, st.LoadAll(ctx))
		require.NoError(t, os.Remove(filepath.Join(root, meetFolder, "LSTSTART.TXT")))

		err := st.LoadAll(ctx)

		require.NoError(t, err)
		assert.Empty(t, st.ListStartEntries(), "a removed file leaves its data empty")
		_, ok := st.GetEvent(1)
		assert.True(t, ok, "the other files are still read")
	})

	t.Run("rejects an invalid file", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTRACE.TXT", "event;round;nbHeat;idLen;idStyle;abCat;date;time\r\nabc;1;1;1;1;\"W\";\"26/05/01\";\"16:00\";\r\n")

		err := NewMeetStore(storage).LoadAll(t.Context())

		require.Error(t, err, "a non-numeric event fails")
		parseErr, ok := errors.AsType[*ParseError](err)
		require.True(t, ok)
		assert.Equal(t, "LSTRACE.TXT", parseErr.File)
		assert.Equal(t, 2, parseErr.Line)
		assert.Equal(t, "event", parseErr.Column)
		assert.ErrorContains(t, err, "LSTRACE.TXT, line 2, column event")
	})

	t.Run("rejects duplicate events", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTRACE.TXT", "event;round;nbHeat;idLen;idStyle;abCat;date;time\r\n"+
			"5;0;1;1;1;\"W\";\"26/05/01\";\"16:00\";\r\n"+
			"5;3;1;1;1;\"W\";\"26/05/01\";\"00:00\";\r\n")

		err := NewMeetStore(storage).LoadAll(t.Context())

		parseErr, ok := errors.AsType[*ParseError](err)
		require.True(t, ok)
		assert.Equal(t, "LSTRACE.TXT", parseErr.File)
		assert.ErrorContains(t, err, "the key 5 occurs more than once")
	})

	t.Run("keeps the previous data after a failed reload", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTRACE.TXT", testEvents)
		writeMeetFile(t, root, "LSTSTART.TXT", testStartEntries)
		st := NewMeetStore(storage)
		require.NoError(t, st.LoadAll(ctx))
		// LSTRACE is read before LSTSTART, so a broken start list fails after the events were parsed
		writeMeetFile(t, root, "LSTRACE.TXT", "event;round;nbHeat;idLen;idStyle;abCat;date;time\r\n2;1;1;1;1;\"W\";\"26/05/01\";\"16:00\";\r\n")
		writeMeetFile(t, root, "LSTSTART.TXT", "event;round;heat;lane;relais;idBib\r\nx;1;0;1;0;1;\r\n")

		err := st.LoadAll(ctx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "LSTSTART.TXT")
		_, ok := st.GetEvent(1)
		assert.True(t, ok, "the previous events stay")
		_, ok = st.GetEvent(2)
		assert.False(t, ok, "the events of the failed reload are not applied")
		assert.Len(t, startsOf(st, 1, 1, 0), 2, "the previous start list stays")
	})
}

// startsOf returns the start entries of an event, phase, and 0-based heat from the index of the store.
func TestParseMeetFiles(t *testing.T) {
	t.Parallel()

	t.Run("reads the meet data of the contents", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		st := NewMeetStore(storage)
		writeMeetFile(t, root, "LSTCONC.TXT", testCompetitors)
		writeMeetFile(t, root, "LSTSTYLE.TXT", testStyles)
		writeMeetFile(t, root, "LSTRACE.TXT", testEvents)
		writeMeetFile(t, root, "LSTSTART.TXT", "event;round;heat;lane;relais;idBib\r\n1;1;0;1;0;7;\r\n")
		require.NoError(t, st.LoadAll(t.Context()))

		got, err := ParseMeetFiles(st.ListFileContents())

		require.NoError(t, err)
		require.Len(t, got.Events, 1)
		require.NotNil(t, got.Events[0].Style)
		assert.Equal(t, "Breaststroke", got.Events[0].Style.Name)
		require.Len(t, got.StartEntries, 1)
		require.NotNil(t, got.StartEntries[0].Competitor, "the references are resolved")
		assert.Equal(t, "Brontë", got.StartEntries[0].Competitor.LastName, "decoded from Windows-1252")
		assert.True(t, got.LoadedAt.IsZero())
	})

	t.Run("names a file that cannot be read", func(t *testing.T) {
		t.Parallel()

		_, err := ParseMeetFiles([]MeetFileContent{
			{
				File:    domain.MeetFileEvents,
				Content: []byte("event;round;nbHeat\nx;y;z\n"),
			},
		})

		parseErr, ok := errors.AsType[*ParseError](err)
		require.True(t, ok)
		assert.Equal(t, string(domain.MeetFileEvents), parseErr.File)
	})
}

func startsOf(st *MeetStore, event, phase, heat int) []*domain.StartEntry {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.data.startsIdx[startsKey(event, phase, heat)]
}
