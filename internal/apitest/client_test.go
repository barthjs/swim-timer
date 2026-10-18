package apitest_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/swim-timer/internal/apitest"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("accepts json", func(t *testing.T) {
		t.Parallel()

		client := apitest.NewClient(t, http.HandlerFunc(echo))

		response := client.GET("/")

		response.AssertJSONPath("accept", "application/json").AssertJSONPath("custom", "")
	})

	t.Run("keeps the cookies the server set", func(t *testing.T) {
		t.Parallel()

		client := apitest.NewClient(t, sessionHandler())
		client.GET("/me").AssertUnauthorized()
		client.POST("/login", nil).AssertOK()

		response := client.GET("/me")

		response.AssertOK().AssertJSON("abc")
		client.WithHeader("X-Other", "1").GET("/me").AssertOK()
	})

	t.Run("keeps the cookies separate per client", func(t *testing.T) {
		t.Parallel()

		handler := sessionHandler()
		apitest.NewClient(t, handler).POST("/login", nil).AssertOK()
		other := apitest.NewClient(t, handler)

		response := other.GET("/me")

		response.AssertUnauthorized()
	})
}

func TestClient_WithHeader(t *testing.T) {
	t.Parallel()

	t.Run("sends the header", func(t *testing.T) {
		t.Parallel()

		client := apitest.NewClient(t, http.HandlerFunc(echo))

		response := client.WithHeader("X-Custom", "1").GET("/")

		response.AssertJSONPath("custom", "1")
	})

	t.Run("keeps the original client unchanged", func(t *testing.T) {
		t.Parallel()

		client := apitest.NewClient(t, http.HandlerFunc(echo))
		client.WithHeader("X-Custom", "1")

		response := client.GET("/")

		response.AssertJSONPath("custom", "")
	})
}

func TestClient_GET(t *testing.T) {
	t.Parallel()

	client := apitest.NewClient(t, http.HandlerFunc(echo))

	response := client.GET("/a")

	response.AssertOK().AssertJSONPath("method", http.MethodGet).AssertJSONPath("path", "/a")
}

func TestClient_POST(t *testing.T) {
	t.Parallel()

	client := apitest.NewClient(t, http.HandlerFunc(echo))

	response := client.POST("/a", nil)

	response.AssertOK().AssertJSONPath("method", http.MethodPost).AssertJSONPath("path", "/a")
}

func TestClient_PUT(t *testing.T) {
	t.Parallel()

	client := apitest.NewClient(t, http.HandlerFunc(echo))

	response := client.PUT("/a", nil)

	response.AssertOK().AssertJSONPath("method", http.MethodPut).AssertJSONPath("path", "/a")
}

func TestClient_PATCH(t *testing.T) {
	t.Parallel()

	client := apitest.NewClient(t, http.HandlerFunc(echo))

	response := client.PATCH("/a", nil)

	response.AssertOK().AssertJSONPath("method", http.MethodPatch).AssertJSONPath("path", "/a")
}

func TestClient_DELETE(t *testing.T) {
	t.Parallel()

	client := apitest.NewClient(t, http.HandlerFunc(echo))

	response := client.DELETE("/a")

	response.AssertOK().AssertJSONPath("method", http.MethodDelete).AssertJSONPath("path", "/a")
}

func TestClient_Do(t *testing.T) {
	t.Parallel()

	t.Run("encodes a json body", func(t *testing.T) {
		t.Parallel()

		client := apitest.NewClient(t, http.HandlerFunc(echo))

		response := client.Do(http.MethodPost, "/", map[string]int{"n": 1})

		response.AssertJSONPath("contentType", "application/json").AssertJSONPath("body", `{"n":1}`)
	})

	t.Run("sends a reader unchanged", func(t *testing.T) {
		t.Parallel()

		client := apitest.NewClient(t, http.HandlerFunc(echo))

		response := client.Do(http.MethodPost, "/", strings.NewReader("raw"))

		response.AssertJSONPath("contentType", "").AssertJSONPath("body", "raw")
	})

	t.Run("encodes a multipart body", func(t *testing.T) {
		t.Parallel()

		client := apitest.NewClient(t, http.HandlerFunc(echo))
		body := apitest.Multipart{
			Fields: map[string]string{"name": "value"},
			Files: []apitest.File{
				{
					Field:   "file",
					Name:    "a.txt",
					Content: []byte("content"),
				},
			},
		}

		response := client.Do(http.MethodPost, "/", body)

		var echoed struct {
			ContentType string `json:"contentType"`
			Body        string `json:"body"`
		}
		response.JSON(&echoed)
		assert.True(t, strings.HasPrefix(echoed.ContentType, "multipart/form-data; boundary="))
		assert.Contains(t, echoed.Body, `filename="a.txt"`)
		assert.Contains(t, echoed.Body, "content")
	})

	t.Run("rejects an unencodable body", func(t *testing.T) {
		t.Parallel()

		mock := &recordingT{
			TB: t,
		}
		client := apitest.NewClient(mock, http.HandlerFunc(echo))

		send := func() { client.Do(http.MethodPost, "/", make(chan int)) }

		require.Panics(t, send)
		assert.True(t, mock.failed)
	})
}

// sessionHandler returns a handler that starts a session at /login and answers /me only with that session.
func sessionHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:  "session",
			Value: "abc",
			Path:  "/",
		})
	})
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`"` + cookie.Value + `"`))
	})

	return mux
}

// echo answers with what it received, so the tests can see how the client built the request.
func echo(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	_ = json.MarshalWrite(w, map[string]any{
		"method":      r.Method,
		"path":        r.URL.Path,
		"contentType": r.Header.Get("Content-Type"),
		"accept":      r.Header.Get("Accept"),
		"custom":      r.Header.Get("X-Custom"),
		"body":        string(body),
	})
}
