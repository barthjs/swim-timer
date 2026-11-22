package store

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
	"github.com/barthjs/swim-timer/internal/apperr"
	"github.com/barthjs/swim-timer/internal/domain"
	"github.com/barthjs/swim-timer/internal/tabular"
)

func TestRecordTableStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns the tables in their order with the time of the load", func(t *testing.T) {
		t.Parallel()

		st := NewRecordTableStore(newTestStorage(t))
		first := testRecordTable(t)
		second := testRecordTable(t)
		require.NoError(t, st.Save(t.Context(), first, []domain.Record{testRecord(t)}))
		require.NoError(t, st.Save(t.Context(), second, nil))

		tables, loadedAt := st.List()

		require.Len(t, tables, 2)
		assert.Equal(t, first, tables[0].Table)
		assert.Len(t, tables[0].Records, 1)
		assert.Equal(t, second, tables[1].Table)
		assert.False(t, loadedAt.IsZero())
	})
}

func TestRecordTableStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns the table with its records", func(t *testing.T) {
		t.Parallel()

		st := NewRecordTableStore(newTestStorage(t))
		table := testRecordTable(t)
		records := []domain.Record{testRecord(t)}
		require.NoError(t, st.Save(t.Context(), table, records))

		data, err := st.Get(table.ID)

		require.NoError(t, err)
		assert.Equal(t, RecordTableData{
			Table:   table,
			Records: records,
		}, data)
	})
}

func TestRecordTableStore_Update(t *testing.T) {
	t.Parallel()

	t.Run("replaces the table and keeps its records", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordTableStore(storage)
		table := testRecordTable(t)
		records := []domain.Record{testRecord(t)}
		require.NoError(t, st.Save(t.Context(), table, records))
		changed := testRecordTable(t)
		changed.ID = table.ID

		err := st.Update(t.Context(), changed)

		require.NoError(t, err)
		reloaded := NewRecordTableStore(storage)
		require.NoError(t, reloaded.LoadAll(t.Context()))
		data, err := reloaded.Get(table.ID)
		require.NoError(t, err)
		assert.Equal(t, changed, data.Table)
		assert.Equal(t, records, data.Records)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()

		st := NewRecordTableStore(newTestStorage(t))

		err := st.Update(t.Context(), testRecordTable(t))

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestRecordTableStore_Save(t *testing.T) {
	t.Parallel()

	t.Run("stores the records in the format of the tables", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordTableStore(storage)
		table := testRecordTable(t)
		record := testRecord(t)
		record.AgeTo = 0

		err := st.Save(t.Context(), table, []domain.Record{record})

		require.NoError(t, err)
		content := readFile(t, storage, recordsFolder+"/"+table.ID+recordTableExtension)
		line := fmt.Sprintf("SCM;M;1;%d;FR;%d;0;%s;%t;%s;%s;%s;%s;%d\r\n", record.Distance, record.AgeFrom,
			FormatSwimTime(record.Time), record.Approved, csvValue(record.Name), csvValue(record.Club), csvValue(record.Place),
			record.Date, record.BirthYear)
		assert.Equal(t, "\xEF\xBB\xBF"+
			"course;gender;relayCount;distance;stroke;ageFrom;ageTo;time;approved;name;club;place;date;birthYear\r\n"+line, string(content))
	})

	t.Run("replaces a table in its place", func(t *testing.T) {
		t.Parallel()

		st := NewRecordTableStore(newTestStorage(t))
		first := testRecordTable(t)
		second := testRecordTable(t)
		require.NoError(t, st.Save(t.Context(), first, nil))
		require.NoError(t, st.Save(t.Context(), second, nil))
		records := []domain.Record{testRecord(t)}

		err := st.Save(t.Context(), first, records)

		require.NoError(t, err)
		tables, _ := st.List()
		require.Len(t, tables, 2)
		assert.Equal(t, first.ID, tables[0].Table.ID)
		assert.Equal(t, records, tables[0].Records)
	})

	t.Run("reads the stored tables before the first change", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		stored := testRecordTable(t)
		require.NoError(t, NewRecordTableStore(storage).Save(t.Context(), stored, []domain.Record{testRecord(t)}))
		st := NewRecordTableStore(storage)
		table := testRecordTable(t)

		err := st.Save(t.Context(), table, nil)

		require.NoError(t, err)
		tables, loadedAt := st.List()
		require.Len(t, tables, 2)
		assert.Equal(t, stored, tables[0].Table)
		assert.Len(t, tables[0].Records, 1)
		assert.Equal(t, table, tables[1].Table)
		assert.False(t, loadedAt.IsZero())
	})

	t.Run("rejects an id that cannot name a file", func(t *testing.T) {
		t.Parallel()

		st := NewRecordTableStore(newTestStorage(t))
		table := testRecordTable(t)
		table.ID = "../x"

		err := st.Save(t.Context(), table, nil)

		assert.ErrorIs(t, err, ErrInvalidName)
	})

	t.Run("rejects a change while a stored table cannot be read and keeps the index", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		stored := testRecordTable(t)
		require.NoError(t, NewRecordTableStore(storage).Save(t.Context(), stored, nil))
		putFile(t, storage, recordsFolder+"/"+stored.ID+recordTableExtension, []byte("course;gender;distance;stroke;time\r\nSCM;M;x;FR;21.75\r\n"))
		index := readFile(t, storage, recordTableIndex)
		st := NewRecordTableStore(storage)
		require.Error(t, st.LoadAll(t.Context()))
		table := testRecordTable(t)

		err := st.Save(t.Context(), table, nil)

		_, ok := errors.AsType[*DataFileError](err)
		assert.True(t, ok, "got %v", err)
		assert.Equal(t, index, readFile(t, storage, recordTableIndex))
		assert.NotContains(t, listFiles(t, storage, recordsFolder+"/"), recordsFolder+"/"+table.ID+recordTableExtension)
		_, loadedAt := st.List()
		assert.True(t, loadedAt.IsZero())
	})

	t.Run("removes the file of a new table if the index cannot be saved", func(t *testing.T) {
		t.Parallel()

		storage := &failingStorage{
			Storage: newTestStorage(t),
			path:    recordTableIndex,
		}
		st := NewRecordTableStore(storage)
		require.NoError(t, st.LoadAll(t.Context()))
		table := testRecordTable(t)

		err := st.Save(t.Context(), table, []domain.Record{testRecord(t)})

		require.ErrorIs(t, err, errStorageFailed)
		assert.Empty(t, listFiles(t, storage, recordsFolder+"/"))
		_, getErr := st.Get(table.ID)
		assert.ErrorIs(t, getErr, apperr.ErrNotFound)
	})
}

func TestRecordTableStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes the table and its file", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordTableStore(storage)
		table := testRecordTable(t)
		require.NoError(t, st.Save(t.Context(), table, nil))

		err := st.Delete(t.Context(), table.ID)

		require.NoError(t, err)
		_, getErr := st.Get(table.ID)
		assert.ErrorIs(t, getErr, apperr.ErrNotFound)
		assert.NotContains(t, listFiles(t, storage, recordsFolder+"/"), recordsFolder+"/"+table.ID+recordTableExtension)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()

		st := NewRecordTableStore(newTestStorage(t))

		err := st.Delete(t.Context(), testRecordTable(t).ID)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})

	t.Run("keeps the table in the index if its file cannot be removed", func(t *testing.T) {
		t.Parallel()

		table := testRecordTable(t)
		storage := &failingStorage{
			Storage: newTestStorage(t),
			path:    recordsFolder + "/" + table.ID + recordTableExtension,
		}
		st := NewRecordTableStore(storage)
		require.NoError(t, st.LoadAll(t.Context()))
		require.NoError(t, NewRecordTableStore(storage.Storage).Save(t.Context(), table, nil))
		require.NoError(t, st.LoadAll(t.Context()))

		err := st.Delete(t.Context(), table.ID)

		require.ErrorIs(t, err, errStorageFailed)
		reloaded := NewRecordTableStore(storage.Storage)
		require.NoError(t, reloaded.LoadAll(t.Context()))
		_, getErr := reloaded.Get(table.ID)
		assert.NoError(t, getErr)
	})
}

func TestRecordTableStore_SaveRecords(t *testing.T) {
	t.Parallel()

	t.Run("replaces the records and keeps the table", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordTableStore(storage)
		table := testRecordTable(t)
		require.NoError(t, st.Save(t.Context(), table, []domain.Record{testRecord(t)}))
		records := []domain.Record{testRecord(t), testRecord(t)}

		saved, err := st.SaveRecords(t.Context(), table.ID, records)

		require.NoError(t, err)
		assert.Equal(t, table, saved)
		reloaded := NewRecordTableStore(storage)
		require.NoError(t, reloaded.LoadAll(t.Context()))
		data, err := reloaded.Get(table.ID)
		require.NoError(t, err)
		assert.Equal(t, records, data.Records)
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()

		st := NewRecordTableStore(newTestStorage(t))

		_, err := st.SaveRecords(t.Context(), testRecordTable(t).ID, nil)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestRecordTableStore_Reorder(t *testing.T) {
	t.Parallel()

	t.Run("puts the tables in the given order", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordTableStore(storage)
		first := testRecordTable(t)
		second := testRecordTable(t)
		require.NoError(t, st.Save(t.Context(), first, nil))
		require.NoError(t, st.Save(t.Context(), second, nil))

		err := st.Reorder(t.Context(), []string{second.ID, first.ID})

		require.NoError(t, err)
		reloaded := NewRecordTableStore(storage)
		require.NoError(t, reloaded.LoadAll(t.Context()))
		tables, _ := reloaded.List()
		require.Len(t, tables, 2)
		assert.Equal(t, second.ID, tables[0].Table.ID)
		assert.Equal(t, first.ID, tables[1].Table.ID)
	})

	invalid := []struct {
		name string
		ids  func(first, second string) []string
	}{
		{
			name: "rejects a missing table",
			ids:  func(first, _ string) []string { return []string{first} },
		},
		{
			name: "rejects a table named twice",
			ids:  func(first, _ string) []string { return []string{first, first} },
		},
		{
			name: "rejects an unknown table",
			ids:  func(first, _ string) []string { return []string{first, "unknown"} },
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			st := NewRecordTableStore(newTestStorage(t))
			first := testRecordTable(t)
			second := testRecordTable(t)
			require.NoError(t, st.Save(t.Context(), first, nil))
			require.NoError(t, st.Save(t.Context(), second, nil))

			err := st.Reorder(t.Context(), tt.ids(first.ID, second.ID))

			assert.ErrorIs(t, err, apperr.ErrNotFound)
			tables, _ := st.List()
			assert.Equal(t, first.ID, tables[0].Table.ID, "the order stays")
		})
	}
}

func TestRecordTableStore_LoadAll(t *testing.T) {
	t.Parallel()

	t.Run("reads the tables of the index with their records", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		table := testRecordTable(t)
		records := []domain.Record{testRecord(t)}
		require.NoError(t, NewRecordTableStore(storage).Save(t.Context(), table, records))
		st := NewRecordTableStore(storage)

		err := st.LoadAll(t.Context())

		require.NoError(t, err)
		data, err := st.Get(table.ID)
		require.NoError(t, err)
		assert.Equal(t, table, data.Table)
		assert.Equal(t, records, data.Records)
	})

	t.Run("adds a file of the folder the index does not list as a disabled table", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, recordsFolder+"/manual.csv", []byte("course;gender;distance;stroke;time\r\nSCM;M;50;FR;21.75\r\n"))
		putFile(t, storage, recordsFolder+"/notes.txt", []byte("x"))
		st := NewRecordTableStore(storage)

		err := st.LoadAll(t.Context())

		require.NoError(t, err)
		tables, _ := st.List()
		require.Len(t, tables, 1)
		assert.Equal(t, domain.RecordTable{
			ID:   "manual",
			Name: "manual",
		}, tables[0].Table)
		require.Len(t, tables[0].Records, 1)
		assert.Equal(t, 2175, tables[0].Records[0].Time)
	})

	t.Run("loads a table without a file as empty", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		table := testRecordTable(t)
		putFile(t, storage, recordTableIndex, []byte(`[{"id":"`+table.ID+`","name":"x","abbreviation":"y","enabled":true}]`))
		st := NewRecordTableStore(storage)

		err := st.LoadAll(t.Context())

		require.NoError(t, err)
		data, err := st.Get(table.ID)
		require.NoError(t, err)
		assert.Empty(t, data.Records)
	})

	t.Run("rejects a file that cannot be read and keeps the previous tables", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		st := NewRecordTableStore(storage)
		table := testRecordTable(t)
		require.NoError(t, st.Save(t.Context(), table, []domain.Record{testRecord(t)}))
		putFile(t, storage, recordsFolder+"/"+table.ID+recordTableExtension, []byte("course;gender;distance;stroke;time\r\nSCM;M;x;FR;21.75\r\n"))

		err := st.LoadAll(t.Context())

		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, table.ID+recordTableExtension, dataErr.File)
		assert.Equal(t, 2, dataErr.Line)
		assert.Equal(t, "distance", dataErr.Column)
		data, getErr := st.Get(table.ID)
		require.NoError(t, getErr)
		assert.Len(t, data.Records, 1)
	})

	t.Run("rejects an index that cannot be read", func(t *testing.T) {
		t.Parallel()

		storage := newTestStorage(t)
		putFile(t, storage, recordTableIndex, []byte("{"))

		err := NewRecordTableStore(storage).LoadAll(t.Context())

		dataErr, ok := errors.AsType[*DataFileError](err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "tables.json", dataErr.File)
	})
}

func TestRecordTableStore_WriteRecords(t *testing.T) {
	t.Parallel()

	t.Run("writes the selected columns in the given format with the formulas escaped", func(t *testing.T) {
		t.Parallel()

		st := NewRecordTableStore(newTestStorage(t))
		table := testRecordTable(t)
		record := testRecord(t)
		record.Name = "=" + record.Name
		require.NoError(t, st.Save(t.Context(), table, []domain.Record{record}))
		var out bytes.Buffer

		err := st.WriteRecords(&out, table.ID, tabular.Format{
			Encoding:  tabular.EncodingWindows1252,
			Separator: ";",
		}, []string{"time", "distance", "name"})

		require.NoError(t, err)
		want := "time;distance;name\r\n" + FormatSwimTime(record.Time) + ";" + strconv.Itoa(record.Distance) + ";'" + csvValue(record.Name) + "\r\n"
		assert.Equal(t, want, out.String())
	})

	t.Run("returns not found for an unknown table", func(t *testing.T) {
		t.Parallel()

		st := NewRecordTableStore(newTestStorage(t))

		err := st.WriteRecords(&bytes.Buffer{}, testRecordTable(t).ID, RecordTableFormat, nil)

		assert.ErrorIs(t, err, apperr.ErrNotFound)
	})
}

func TestNewRecordSchema(t *testing.T) {
	t.Parallel()

	t.Run("reads a table of another program through a mapping", func(t *testing.T) {
		t.Parallel()

		schema := NewRecordSchema(RecordSchemaOptions{
			Genders: []string{"M", "W"},
			Strokes: []string{"FR", "BA"},
		})
		faker := apitest.Faker(t)
		name := faker.LastName()
		date := time.Date(faker.IntRange(1990, 2025), time.Month(faker.IntRange(1, 12)), faker.IntRange(1, 28), 0, 0, 0, 0, time.UTC)
		birthYear := faker.IntRange(1970, 2015)
		content := "Laenge;Lage;Geschlecht;Bahn;VonAlter;BisAlter;Zeit;Anerkannt;Name;Datum;Jg\r\n" +
			fmt.Sprintf("100;F;w;Lang;0;0;00:58,12;Ja;%s, ;%s;%d\r\n", name, date.Format("02.01.2006"), birthYear)
		preview, err := schema.Preview(strings.NewReader(content), tabular.Format{})
		require.NoError(t, err)
		mapping := preview.Mapping
		require.Equal(t, map[string]string{
			"Lang": "LCM",
		}, mapping.Values["course"], "the German name of the course is guessed")
		require.Equal(t, map[string]string{
			"Ja": "true",
		}, mapping.Values["approved"], "the German word for yes is guessed")
		require.Equal(t, map[string]string{
			"F": "FR",
		}, mapping.Values["stroke"], "the letter of the style is guessed")

		imp, err := schema.Read(strings.NewReader(content), mapping)
		require.NoError(t, err)
		rows, failed, err := tabular.Collect(imp.Rows(t.Context()), 10)

		require.NoError(t, err)
		require.Empty(t, failed)
		require.Len(t, rows, 1)
		assert.Equal(t, &domain.Record{
			Course:     domain.CourseLong,
			Gender:     "W",
			RelayCount: 1,
			Distance:   100,
			Stroke:     "FR",
			Time:       5812,
			Approved:   true,
			Name:       name,
			Date:       date.Format(time.DateOnly),
			BirthYear:  birthYear,
		}, rows[0].Value, "a relay count defaults to an individual record")
	})

	t.Run("suggests the styles of the catalog without meet data", func(t *testing.T) {
		t.Parallel()

		schema := NewRecordSchema(RecordSchemaOptions{})

		columns := schema.Columns()

		for _, column := range columns {
			if column.Key == "stroke" {
				assert.Equal(t, []string{"BA", "BR", "BU", "FR", "ME"}, column.Suggestions)
			}
		}
	})

	t.Run("rejects an upper age below the lower one", func(t *testing.T) {
		t.Parallel()

		schema := NewRecordSchema(RecordSchemaOptions{})
		content := "course;gender;distance;stroke;time;ageFrom;ageTo\r\nSCM;M;50;FR;21.75;14;12\r\n"

		imp, err := schema.Read(strings.NewReader(content), tabular.Mapping{
			Columns: map[string]string{
				"course":   "course",
				"gender":   "gender",
				"distance": "distance",
				"stroke":   "stroke",
				"time":     "time",
				"ageFrom":  "ageFrom",
				"ageTo":    "ageTo",
			},
		})
		require.NoError(t, err)
		_, failed, err := tabular.Collect(imp.Rows(t.Context()), 10)

		require.NoError(t, err)
		require.Len(t, failed, 1)
		assert.Equal(t, "gtefield", failed[0].Fields["ageTo"][0].Tag)
	})
}

func TestParseSwimTime(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name string
		raw  string
		want int
	}{
		{
			name: "reads minutes and seconds with a comma",
			raw:  "07:32,12",
			want: 45212,
		},
		{
			name: "reads minutes and seconds with a point",
			raw:  "1:02.34",
			want: 6234,
		},
		{
			name: "reads seconds only",
			raw:  "58.1",
			want: 5810,
		},
		{
			name: "reads hours",
			raw:  "1:00:00.00",
			want: 360000,
		},
		{
			name: "reads whole seconds",
			raw:  "45",
			want: 4500,
		},
	}

	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hundredths, err := ParseSwimTime(tt.raw)

			require.NoError(t, err)
			assert.Equal(t, tt.want, hundredths)
		})
	}

	invalid := []struct {
		name string
		raw  string
	}{
		{
			name: "rejects text",
			raw:  "x",
		},
		{
			name: "rejects thousandths",
			raw:  "58.123",
		},
		{
			name: "rejects seconds of 60 after minutes",
			raw:  "1:60.00",
		},
		{
			name: "rejects minutes of 60 after hours",
			raw:  "1:60:00.00",
		},
		{
			name: "rejects more than three parts",
			raw:  "1:1:1:1",
		},
		{
			name: "rejects an empty part",
			raw:  ":58.00",
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseSwimTime(tt.raw)

			assert.Equal(t, &tabular.FieldError{
				Tag: "swimtime",
			}, err)
		})
	}
}

func TestFormatSwimTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		hundredths int
		want       string
	}{
		{
			name:       "writes minutes, seconds, and hundredths",
			hundredths: 6234,
			want:       "01:02.34",
		},
		{
			name:       "writes the hours of a long time",
			hundredths: 372345,
			want:       "1:02:03.45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			text := FormatSwimTime(tt.hundredths)

			assert.Equal(t, tt.want, text)
			parsed, err := ParseSwimTime(text)
			require.NoError(t, err)
			assert.Equal(t, tt.hundredths, parsed, "the text reads back")
		})
	}
}

// csvValue quotes a value of an expected CSV row as the writer does.
func csvValue(value string) string {
	if strings.ContainsAny(value, ";\"\r\n") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return value
}

// testRecordTable returns a record table with random values.
func testRecordTable(t *testing.T) domain.RecordTable {
	t.Helper()

	faker := apitest.Faker(t)
	return domain.RecordTable{
		ID:           strings.ToLower(faker.LetterN(12)),
		Name:         faker.LetterN(12),
		Abbreviation: faker.LetterN(2),
		Enabled:      faker.Bool(),
	}
}
