package service

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/store"
	"github.com/barthjs/swim-timer/internal/tabular"
)

func TestRecordTableService_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the tables in their order with the number of their records", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		first := createTestRecordTable(t, svc)
		second := createTestRecordTable(t, svc)

		list := svc.List()

		assert.Equal(t, []RecordTableState{
			first,
			second,
		}, list.Tables)
		assert.Equal(t, 2, list.Tables[0].RecordCount)
		assert.False(t, list.LoadedAt.IsZero())
	})
}

func TestRecordTableService_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the table", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		created := createTestRecordTable(t, svc)

		table, err := svc.Get(created.Table.ID)

		require.NoError(t, err)
		assert.Equal(t, created, table)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()

		_, err := newTestRecordTableService(t).Get(apitest.Faker(t).LetterN(8))

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestRecordTableService_Create(t *testing.T) {
	t.Parallel()

	t.Run("imports the records of the file through the mapping", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		name := apitest.Faker(t).LetterN(12)
		content, holders := newTestRecordFile(t)

		table, err := svc.Create(t.Context(), RecordTableCreateInput{
			Name:         name,
			Abbreviation: "MR",
			Enabled:      true,
			Content:      []byte(content),
			Mapping:      testRecordMapping(t, svc),
		})

		require.NoError(t, err)
		assert.Equal(t, name, table.Table.Name)
		assert.NotEmpty(t, table.Table.ID)
		assert.Equal(t, 2, table.RecordCount)
		records := svc.ListRecords().Records
		require.Len(t, records, 2)
		assert.Equal(t, domain.Record{
			Course:     domain.CourseShort,
			Gender:     "O",
			RelayCount: 1,
			Distance:   50,
			Stroke:     "BA",
			AgeFrom:    12,
			AgeTo:      14,
			Time:       3010,
			Name:       holders[1].Name,
			Club:       holders[1].Club,
			Place:      holders[1].Place,
			Date:       holders[1].Date.Format(time.DateOnly),
			BirthYear:  holders[1].BirthYear,
		}, records[1].Record)
	})

	t.Run("rejects a file that cannot be read", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)

		_, err := svc.Create(t.Context(), RecordTableCreateInput{
			Name:         apitest.Faker(t).LetterN(12),
			Abbreviation: "MR",
		})

		assertValidation(t, err, "file", "empty")
	})

	t.Run("rejects a mapping without a required column", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		mapping := testRecordMapping(t, svc)
		delete(mapping.Columns, "time")

		_, err := svc.Create(t.Context(), RecordTableCreateInput{
			Name:         apitest.Faker(t).LetterN(12),
			Abbreviation: "MR",
			Content:      []byte(testRecordContent(t)),
			Mapping:      mapping,
		})

		assertValidation(t, err, "mapping.columns.time", "required")
	})

	t.Run("rejects the rows that fail and stores nothing", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		mapping := testRecordMapping(t, svc)
		delete(mapping.Values["course"], "Kurz")

		_, err := svc.Create(t.Context(), RecordTableCreateInput{
			Name:         apitest.Faker(t).LetterN(12),
			Abbreviation: "MR",
			Content:      []byte(testRecordContent(t)),
			Mapping:      mapping,
		})

		assertValidation(t, err, "rows.3.course", "oneof")
		assert.Empty(t, svc.List().Tables)
	})

	t.Run("reports a row that ends early as a failure of the row", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)

		_, err := svc.Create(t.Context(), RecordTableCreateInput{
			Name:         apitest.Faker(t).LetterN(12),
			Abbreviation: "MR",
			Content:      []byte(testRecordContent(t) + "1;100\r\n"),
			Mapping:      testRecordMapping(t, svc),
		})

		assertValidation(t, err, "rows.4", "short_row")
	})
}

func TestRecordTableService_Update(t *testing.T) {
	t.Parallel()

	t.Run("replaces the name, the abbreviation, and the state", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		created := createTestRecordTable(t, svc)
		name := apitest.Faker(t).LetterN(12)

		table, err := svc.Update(t.Context(), created.Table.ID, RecordTableUpdateInput{
			Name:         name,
			Abbreviation: "VR",
			Enabled:      false,
		})

		require.NoError(t, err)
		assert.Equal(t, RecordTableState{
			Table: domain.RecordTable{
				ID:           created.Table.ID,
				Name:         name,
				Abbreviation: "VR",
			},
			RecordCount: 2,
		}, table)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()

		_, err := newTestRecordTableService(t).Update(t.Context(), apitest.Faker(t).LetterN(8), RecordTableUpdateInput{})

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestRecordTableService_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the table", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		created := createTestRecordTable(t, svc)

		err := svc.Delete(t.Context(), created.Table.ID)

		require.NoError(t, err)
		assert.Empty(t, svc.List().Tables)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()

		err := newTestRecordTableService(t).Delete(t.Context(), apitest.Faker(t).LetterN(8))

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestRecordTableService_ListRecords(t *testing.T) {
	t.Parallel()

	t.Run("returns the records of every table with their table", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		first := createTestRecordTable(t, svc)
		second := createTestRecordTable(t, svc)

		list := svc.ListRecords()

		require.Len(t, list.Records, 4)
		assert.Equal(t, first.Table, list.Records[0].Table)
		assert.Equal(t, second.Table, list.Records[3].Table)
	})

	t.Run("returns no records without tables", func(t *testing.T) {
		t.Parallel()

		list := newTestRecordTableService(t).ListRecords()

		assert.Empty(t, list.Records)
	})
}

func TestRecordTableService_Preview(t *testing.T) {
	t.Parallel()

	t.Run("guesses the mapping and suggests the categories and styles of the meet", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)

		preview, err := svc.Preview([]byte(testRecordContent(t)), tabular.Format{})

		require.NoError(t, err)
		assert.Equal(t, "Zeit", preview.Preview.Mapping.Columns["time"])
		assert.Equal(t, "Lage", preview.Preview.Mapping.Columns["stroke"])
		assert.Equal(t, map[string]string{
			"O": "O",
		}, preview.Preview.Mapping.Values["gender"])
		var stroke tabular.ColumnInfo
		for _, column := range preview.Columns {
			if column.Key == "stroke" {
				stroke = column
			}
		}
		assert.Equal(t, []string{"BA"}, stroke.Suggestions)
	})

	invalid := []struct {
		name    string
		content []byte
		tag     string
	}{
		{
			name: "rejects an empty file",
			tag:  "empty",
		},
		{
			name:    "rejects a file in utf-16",
			content: []byte{0xFF, 0xFE, 'a', 0},
			tag:     "encoding",
		},
		{
			name:    "rejects a file with a quote that is not closed with its line",
			content: []byte("Zeit;Name\r\n1:00,00;\"x\r\n"),
			tag:     "unreadable_line",
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := newTestRecordTableService(t).Preview(tt.content, tabular.Format{})

			assertValidation(t, err, "file", tt.tag)
		})
	}
}

func TestRecordTableService_Import(t *testing.T) {
	t.Parallel()

	t.Run("replaces the records of the table", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		created := createTestRecordTable(t, svc)
		lines := "Laenge;Lage;Geschlecht;Bahn;Zeit\r\n200;R;O;Lang;2:10,00\r\n"
		mapping := testRecordMapping(t, svc)
		mapping.Columns = map[string]string{
			"distance": "Laenge",
			"stroke":   "Lage",
			"gender":   "Geschlecht",
			"course":   "Bahn",
			"time":     "Zeit",
		}

		table, err := svc.Import(t.Context(), created.Table.ID, RecordTableImportInput{
			Content: []byte(lines),
			Mapping: mapping,
		})

		require.NoError(t, err)
		assert.Equal(t, 1, table.RecordCount)
		assert.Equal(t, created.Table, table.Table)
		records := svc.ListRecords().Records
		require.Len(t, records, 1)
		assert.Equal(t, 13000, records[0].Record.Time)
	})

	t.Run("keeps the records on failure", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		created := createTestRecordTable(t, svc)

		_, err := svc.Import(t.Context(), created.Table.ID, RecordTableImportInput{
			Content: []byte("x\r\n"),
			Mapping: testRecordMapping(t, svc),
		})

		require.Error(t, err)
		assert.Len(t, svc.ListRecords().Records, 2)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)

		_, err := svc.Import(t.Context(), apitest.Faker(t).LetterN(8), RecordTableImportInput{
			Content: []byte(testRecordContent(t)),
			Mapping: testRecordMapping(t, svc),
		})

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestRecordTableService_Reorder(t *testing.T) {
	t.Parallel()

	t.Run("puts the tables in the given order", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		first := createTestRecordTable(t, svc)
		second := createTestRecordTable(t, svc)

		err := svc.Reorder(t.Context(), []string{second.Table.ID, first.Table.ID})

		require.NoError(t, err)
		tables := svc.List().Tables
		assert.Equal(t, second.Table.ID, tables[0].Table.ID)
	})

	t.Run("rejects ids that do not name every table once", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		first := createTestRecordTable(t, svc)
		createTestRecordTable(t, svc)

		err := svc.Reorder(t.Context(), []string{first.Table.ID})

		assertValidation(t, err, "ids", "invalid")
	})
}

func TestRecordTableService_Reload(t *testing.T) {
	t.Parallel()

	t.Run("reads a table changed in the storage", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		svc := NewRecordTableService(store.NewRecordTableStore(storage), newTestMeetStore(t, tenLaneFiles))
		content := "course;gender;distance;stroke;time\r\nSCM;O;50;BA;30.00\r\n"
		require.NoError(t, storage.Put(t.Context(), "records/manual.csv", strings.NewReader(content), -1))

		err := svc.Reload(t.Context())

		require.NoError(t, err)
		assert.Len(t, svc.ListRecords().Records, 1)
	})

	t.Run("names a table that cannot be read", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		svc := NewRecordTableService(store.NewRecordTableStore(storage), newTestMeetStore(t, tenLaneFiles))
		content := "course;gender;distance;stroke;time\r\nSCM;O;x;BA;30.00\r\n"
		require.NoError(t, storage.Put(t.Context(), "records/manual.csv", strings.NewReader(content), -1))

		err := svc.Reload(t.Context())

		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "manual.csv", dataErr.File)
	})
}

func TestRecordTableService_Export(t *testing.T) {
	t.Parallel()

	t.Run("writes the selected columns in the given format", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		created := createTestRecordTable(t, svc)
		var out bytes.Buffer

		err := svc.Export(&out, created.Table.ID, RecordTableExportInput{
			Format: tabular.Format{
				Encoding:  tabular.EncodingWindows1252,
				Separator: "\t",
			},
			Columns: []string{"distance", "time"},
		})

		require.NoError(t, err)
		assert.Equal(t, "distance\ttime\r\n100\t01:01.00\r\n50\t00:30.10\r\n", out.String())
	})

	t.Run("rejects an unknown column", func(t *testing.T) {
		t.Parallel()

		svc := newTestRecordTableService(t)
		created := createTestRecordTable(t, svc)

		err := svc.Export(&bytes.Buffer{}, created.Table.ID, RecordTableExportInput{
			Columns: []string{"other"},
		})

		assertValidation(t, err, "columns", "oneof")
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()

		err := newTestRecordTableService(t).Export(&bytes.Buffer{}, apitest.Faker(t).LetterN(8), RecordTableExportInput{})

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

// newTestRecordTableService creates a record table service with the meet of a pool with ten lanes,
// whose category is "O" and whose style is "BA".
func newTestRecordTableService(t *testing.T) *RecordTableService {
	t.Helper()

	return NewRecordTableService(store.NewRecordTableStore(newTestStorage(t)), newTestMeetStore(t, tenLaneFiles))
}

// createTestRecordTable creates an enabled record table with the records of a file of newTestRecordFile.
func createTestRecordTable(t *testing.T, svc *RecordTableService) RecordTableState {
	t.Helper()

	faker := apitest.Faker(t)
	table, err := svc.Create(t.Context(), RecordTableCreateInput{
		Name:         faker.LetterN(12),
		Abbreviation: faker.LetterN(2),
		Enabled:      true,
		Content:      []byte(testRecordContent(t)),
		Mapping:      testRecordMapping(t, svc),
	})
	require.NoError(t, err)
	return table
}

// testRecordMapping returns the mapping of a file of newTestRecordFile as the preview guesses it,
// which translates every value.
func testRecordMapping(t *testing.T, svc *RecordTableService) tabular.Mapping {
	t.Helper()

	preview, err := svc.Preview([]byte(testRecordContent(t)), tabular.Format{})
	require.NoError(t, err)
	return preview.Preview.Mapping
}

// testRecordContent returns the content of a file of newTestRecordFile.
func testRecordContent(t *testing.T) string {
	t.Helper()

	content, _ := newTestRecordFile(t)
	return content
}

// testRecordHolder is the holder of a record of a file of newTestRecordFile.
type testRecordHolder struct {
	Name      string
	Club      string
	Place     string
	Date      time.Time
	BirthYear int
}

// newTestRecordFile returns a table of two records as another program exports it, with German headers and values,
// and the holders of its records: an open approved record over 100 m in a 50 m pool,
// and a pending record over 50 m in a 25 m pool for the ages 12 to 14, both backstroke of the category "O".
func newTestRecordFile(t *testing.T) (string, [2]testRecordHolder) {
	t.Helper()

	faker := apitest.Faker(t)
	var holders [2]testRecordHolder
	for i := range holders {
		holders[i] = testRecordHolder{
			Name:      faker.LastName() + ", " + faker.FirstName(),
			Club:      faker.LetterN(10),
			Place:     faker.LetterN(8),
			Date:      time.Date(faker.IntRange(1990, 2025), time.Month(faker.IntRange(1, 12)), faker.IntRange(1, 28), 0, 0, 0, 0, time.UTC),
			BirthYear: faker.IntRange(1970, 2015),
		}
	}

	content := "Anzahl;Laenge;Lage;Geschlecht;Bahn;VonAlter;BisAlter;Zeit;Anerkannt;Name;Verein;Ort;Datum;Jg;\r\n"
	rows := []string{
		"1;100;R;O;Lang;0;0;01:01,00;Ja",
		"1;50;R;O;Kurz;12;14;00:30,10;Nein",
	}
	for i, row := range rows {
		holder := holders[i]
		date := holder.Date.Format("02.01.2006")
		content += fmt.Sprintf("%s;%s;%s;%s;%s;%d;\r\n", row, holder.Name, holder.Club, holder.Place, date, holder.BirthYear)
	}
	return content, holders
}
