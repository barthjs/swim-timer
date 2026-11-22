package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/media"
	"github.com/barthjs/swim-timer/internal/tabular"
)

const (
	testStyles       = "idStyle;Style;StyleAbrév\r\n1;\"Breaststroke\";\"BR\"\r\n"
	testCompetitors  = "id;bib;lastname;firstname;birthyear;abNat;abCat\r\n7;;\"Brontë\";\"Chloé\";\"2010\";\"SC South\";\"M\";\r\n"
	testEvents       = "event;round;nbHeat;idLen;idStyle;abCat;date;time\r\n1;1;1;1;1;\"W\";\"26/05/01\";\"16:00\";\r\n"
	testStartEntries = "event;round;heat;lane;relais;idBib\r\n1;1;0;1;0;1;\r\n1;1;0;2;0;2;\r\n"
)

func TestMeetStore_GetHeat(t *testing.T) {
	t.Parallel()

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
		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok)
		assert.Equal(t, "LSTRACE.TXT", dataErr.File)
		assert.Equal(t, 2, dataErr.Line)
		assert.Equal(t, "event", dataErr.Column)
		assert.ErrorContains(t, err, "LSTRACE.TXT, line 2, column event")
	})

	t.Run("rejects duplicate events", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "LSTRACE.TXT", "event;round;nbHeat;idLen;idStyle;abCat;date;time\r\n"+
			"5;0;1;1;1;\"W\";\"26/05/01\";\"16:00\";\r\n"+
			"5;3;1;1;1;\"W\";\"26/05/01\";\"00:00\";\r\n")

		err := NewMeetStore(storage).LoadAll(t.Context())

		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok)
		assert.Equal(t, "LSTRACE.TXT", dataErr.File)
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

		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok)
		assert.Equal(t, string(domain.MeetFileEvents), dataErr.File)
	})
}

// startsOf returns the start entries of a heat from the index of the loaded meet data.
func startsOf(st *MeetStore, event, phase, heat int) []*domain.StartEntry {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.data.startsIdx[startsKey(event, phase, heat)]
}

// csvRow is a row type with the column layout of a small export file.
type csvRow struct {
	ID   int
	Name string
}

// csvRowSchema reads the columns of csvRow.
var csvRowSchema = tabular.New([]tabular.Column[csvRow]{
	meetInt("id", func(r *csvRow, v int) { r.ID = v }),
	meetString("name", func(r *csvRow, v string) { r.Name = v }),
})

func TestReadMeetFile(t *testing.T) {
	t.Parallel()

	t.Run("returns the content as it is stored", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		content := []byte{'i', 'd', 0xE4, '\r', '\n'}
		dir := filepath.Join(root, meetFolder)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "TEST.TXT"), content, 0o644))

		got, err := readMeetFile(t.Context(), storage, meetFolder+"/TEST.TXT")

		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("fails for a missing file", func(t *testing.T) {
		t.Parallel()

		storage, _ := newTestStorageWithRoot(t)

		_, err := readMeetFile(t.Context(), storage, meetFolder+"/TEST.TXT")

		assert.Error(t, err)
	})
}

func TestLoadMap(t *testing.T) {
	t.Parallel()

	t.Run("rejects a file that cannot be read", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "TEST.TXT", "id;name\r\n1;x;\r\n2\r\n")

		err := loadMap(readTestFile(t, storage), csvRowSchema, map[int]*csvRow{}, func(r *csvRow) int { return r.ID })

		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok)
		assert.Equal(t, 3, dataErr.Line)
	})

	t.Run("rejects a duplicate key with its line in the file", func(t *testing.T) {
		t.Parallel()

		storage, root := newTestStorageWithRoot(t)
		writeMeetFile(t, root, "TEST.TXT", "id;name\r\n1;x;\r\n\r\n2;y;\r\n1;z;\r\n")

		err := loadMap(readTestFile(t, storage), csvRowSchema, map[int]*csvRow{}, func(r *csvRow) int { return r.ID })

		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok)
		assert.Equal(t, 5, dataErr.Line, "the line in the file, counting the skipped blank row")
		assert.ErrorContains(t, err, "line 5")
	})
}

func TestReadCSV(t *testing.T) {
	t.Parallel()

	t.Run("returns the rows of a file as the timing software writes it", func(t *testing.T) {
		t.Parallel()

		name := apitest.Faker(t).LetterN(4) + "äß"

		rows, err := readTestCSV(t, "id;name;\r\n1;\""+name+"\";\r\n \r\n;;\r\n2;x\"y;\r\n")

		require.NoError(t, err)
		require.Len(t, rows, 2, "rows without any value are skipped")
		assert.Equal(t, name, rows[0].Name, "the special characters of windows-1252 are decoded")
		assert.Equal(t, 2, rows[1].ID)
		assert.Equal(t, "x\"y", rows[1].Name, "a stray quote inside a value is kept")
	})

	t.Run("reads utf-8 whose special characters come late in the file", func(t *testing.T) {
		t.Parallel()

		name := apitest.Faker(t).LetterN(4) + "äß"
		content := "id;name\r\n" + strings.Repeat("1;x;\r\n", 20_000) + "2;\"" + name + "\";\r\n"

		rows, err := readRawTestCSV(t, []byte(content))

		require.NoError(t, err)
		assert.Equal(t, name, rows[len(rows)-1].Name, "the encoding is detected from the whole file")
	})

	invalid := []struct {
		name     string
		content  string
		wantLine int
		wantErr  string
	}{
		{
			name:     "rejects a missing column",
			content:  "id;title\r\n1;x\r\n",
			wantLine: 1,
			wantErr:  "lacks the columns name",
		},
		{
			name:     "rejects a cut off row",
			content:  "id;name\r\n1;x;\r\n2\r\n",
			wantLine: 3,
			wantErr:  "1 of 2 values",
		},
		{
			name:     "rejects a quote that is not closed",
			content:  "id;name\r\n1;x;\r\n2;\"y;\r\n3;z;\r\n4;w;\r\n",
			wantLine: 3,
			wantErr:  "column name: a quote is not closed",
		},
		{
			name:     "rejects an invalid number",
			content:  "id;name\r\n1;x\r\n\r\nx;y\r\n",
			wantLine: 4,
			wantErr:  `column id: the value "x" is not a number`,
		},
		{
			name:    "rejects an empty file",
			content: "",
			wantErr: "empty",
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := readTestCSV(t, tt.content)

			dataErr, ok := errors.AsType[*DataFileError](err)
			require.True(t, ok)
			assert.Equal(t, tt.wantLine, dataErr.Line)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}

	t.Run("rejects a utf-16 byte order mark", func(t *testing.T) {
		t.Parallel()

		content := append([]byte{0xFE, 0xFF}, []byte("id;name\r\n1;x\r\n")...)

		_, err := readRawTestCSV(t, content)

		_, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok)
		assert.ErrorContains(t, err, "UTF-16")
	})
}

// readTestCSV writes content as a Windows-1252 meet file and reads it into rows.
func readTestCSV(t *testing.T, content string) ([]*csvRow, error) {
	t.Helper()

	storage, root := newTestStorageWithRoot(t)
	writeMeetFile(t, root, "TEST.TXT", content)

	rows, _, err := readCSV(readTestFile(t, storage), csvRowSchema)
	return rows, err
}

// readRawTestCSV writes content unencoded as a meet file and reads it into rows.
func readRawTestCSV(t *testing.T, content []byte) ([]*csvRow, error) {
	t.Helper()

	storage, root := newTestStorageWithRoot(t)
	dir := filepath.Join(root, meetFolder)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "TEST.TXT"), content, 0o644))

	rows, _, err := readCSV(readTestFile(t, storage), csvRowSchema)
	return rows, err
}

// readTestFile reads the test meet file from the storage.
func readTestFile(t *testing.T, storage media.Storage) []byte {
	t.Helper()

	content, err := readMeetFile(t.Context(), storage, meetFolder+"/TEST.TXT")
	require.NoError(t, err)
	return content
}

func TestFillMap(t *testing.T) {
	t.Parallel()

	t.Run("rejects a duplicate key", func(t *testing.T) {
		t.Parallel()

		rows := []*csvRow{
			{
				ID: 1,
			},
			{
				ID: 1,
			},
		}

		err := fillMap(rows, []int{2, 5}, map[int]*csvRow{}, func(r *csvRow) int { return r.ID })

		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok)
		assert.Equal(t, 5, dataErr.Line, "the line of the second occurrence")
		assert.ErrorContains(t, err, "the key 1 occurs more than once")
	})
}
