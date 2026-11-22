package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestQueryParser_Optional(t *testing.T) {
	t.Parallel()

	t.Run("returns the trimmed value through the parse function", func(t *testing.T) {
		t.Parallel()
		value := apitest.Faker(t).IntRange(-1000, 1000)
		p := newQueryParser(queryRequest(t, url.Values{"key": {" " + strconv.Itoa(value) + " "}}))

		got := p.optional("key", strconv.Atoi)

		require.NoError(t, p.err)
		require.NotNil(t, got)
		assert.Equal(t, value, *got)
	})

	t.Run("returns nil for a missing or blank value", func(t *testing.T) {
		t.Parallel()
		p := newQueryParser(queryRequest(t, url.Values{"blank": {" "}}))

		missing := p.optional("missing", strconv.Atoi)
		blank := p.optional("blank", strconv.Atoi)

		require.NoError(t, p.err)
		assert.Nil(t, missing)
		assert.Nil(t, blank)
	})

	t.Run("rejects a malformed value with the name of the parameter", func(t *testing.T) {
		t.Parallel()
		p := newQueryParser(queryRequest(t, url.Values{"key": {apitest.Faker(t).LetterN(8)}}))

		got := p.optional("key", strconv.Atoi)

		assert.Nil(t, got)
		assert.EqualError(t, p.err, "invalid query parameter key")
	})
}

func TestQueryParser_Fail(t *testing.T) {
	t.Parallel()

	t.Run("keeps the first error", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		values := url.Values{
			"first":  {f.LetterN(8)},
			"second": {f.LetterN(8)},
		}
		p := newQueryParser(queryRequest(t, values))

		p.optionalInt("first")
		p.optionalBool("second")

		assert.EqualError(t, p.err, "invalid query parameter first")
	})

	t.Run("skips the reads after an error", func(t *testing.T) {
		t.Parallel()
		f := apitest.Faker(t)
		values := url.Values{
			"invalid": {f.LetterN(8)},
			"valid":   {f.Word()},
		}
		p := newQueryParser(queryRequest(t, values))

		p.optionalInt("invalid")
		got := p.optionalString("valid")

		assert.Nil(t, got)
	})
}

func TestPathInt(t *testing.T) {
	t.Parallel()

	t.Run("returns the number", func(t *testing.T) {
		t.Parallel()
		number := apitest.Faker(t).IntRange(1, 1000)

		got, err := pathInt(pathParamRequest(t, "number", strconv.Itoa(number)), "number")

		require.NoError(t, err)
		assert.Equal(t, number, got)
	})

	t.Run("rejects an invalid value", func(t *testing.T) {
		t.Parallel()

		f := apitest.Faker(t)
		tests := []struct {
			name string
			raw  string
			want string
		}{
			{
				name: "missing",
				raw:  "",
				want: "missing path parameter number",
			},
			{
				name: "malformed",
				raw:  f.LetterN(8),
				want: "invalid path parameter number",
			},
			{
				name: "zero",
				raw:  "0",
				want: "invalid path parameter number",
			},
			{
				name: "negative",
				raw:  strconv.Itoa(-f.IntRange(1, 1000)),
				want: "invalid path parameter number",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := pathInt(pathParamRequest(t, "number", tt.raw), "number")

				assert.EqualError(t, err, tt.want)
			})
		}
	})
}

func TestPathParam(t *testing.T) {
	t.Parallel()

	t.Run("decodes a parameter the browser escapes differently than Go", func(t *testing.T) {
		t.Parallel()
		var got string
		router := chi.NewRouter()
		router.Get("/{name}", func(_ http.ResponseWriter, r *http.Request) { got = pathParam(r, "name") })

		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/Single%20image%20(10%20x%2015%20cm)", nil))

		assert.Equal(t, "Single image (10 x 15 cm)", got)
	})
}

func TestParseBool(t *testing.T) {
	t.Parallel()

	t.Run("returns the value", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			raw  string
			want bool
		}{
			{
				raw:  "true",
				want: true,
			},
			{
				raw:  "1",
				want: true,
			},
			{
				raw:  "Yes",
				want: true,
			},
			{
				raw:  "false",
				want: false,
			},
			{
				raw:  "0",
				want: false,
			},
			{
				raw:  "No",
				want: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.raw, func(t *testing.T) {
				t.Parallel()

				got, err := parseBool(tt.raw)

				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("rejects a malformed value", func(t *testing.T) {
		t.Parallel()

		_, err := parseBool(apitest.Faker(t).LetterN(8))

		assert.Error(t, err)
	})
}

// pathParamRequest builds a request whose route holds the path parameter name with value.
func pathParamRequest(t *testing.T, name, value string) *http.Request {
	t.Helper()

	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(name, value)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

// queryRequest builds a request whose query string holds the values.
func queryRequest(t *testing.T, values url.Values) *http.Request {
	t.Helper()

	return httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/?"+values.Encode(), nil)
}
