package huma

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRecoveryMiddleware(t *testing.T) {
	r := NewTestRouter(t)
	r.GinEngine().Use(Recovery())

	r.Resource("/panic").Get("Panic recovery test", func() string {
		panic(fmt.Errorf("Some error"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/problem+json", w.Result().Header.Get("content-type"))
}

func TestRecoveryMiddlewareString(t *testing.T) {
	r := NewTestRouter(t)
	r.GinEngine().Use(Recovery())

	r.Resource("/panic").Get("Panic recovery test", func() string {
		panic("Some error")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/problem+json", w.Result().Header.Get("content-type"))
}

func TestRecoveryMiddlewareLogBody(t *testing.T) {
	r := NewTestRouter(t)
	r.GinEngine().Use(Recovery())

	r.Resource("/panic").Put("Panic recovery test", func(in map[string]string) string {
		panic(fmt.Errorf("Some error"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/panic", strings.NewReader(`{"foo": "bar"}`))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/problem+json", w.Result().Header.Get("content-type"))
}

func TestPreferMinimalMiddleware(t *testing.T) {
	r := NewTestRouter(t)
	r.GinEngine().Use(PreferMinimalMiddleware())

	r.Resource("/test").Get("desc", func() string {
		return "Hello, test"
	})

	r.Resource("/non200", ResponseText(http.StatusBadRequest, "desc")).Get("desc", func() string {
		return "Error details"
	})

	// Normal request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())

	// Prefer minimal should return 204 No Content
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Add("prefer", "return=minimal")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())

	// Prefer minimal which can still return non-200 response bodies
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/non200", nil)
	req.Header.Add("prefer", "return=minimal")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestHandler404(t *testing.T) {
	g := gin.New()
	g.NoRoute(Handler404())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notfound", nil)
	g.ServeHTTP(w, req)
	assert.Equal(t, w.Result().StatusCode, http.StatusNotFound)
	assert.Equal(t, "application/problem+json", w.Result().Header.Get("content-type"))
}

func TestServiceLinks(t *testing.T) {
	r := NewTestRouter(t)
	r.GinEngine().Use(ServiceLinkMiddleware())
	r.GinEngine().NoRoute(Handler404())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Result().StatusCode, http.StatusNoContent)
	assert.Contains(t, w.Result().Header.Get("link"), "service-desc")
	assert.Contains(t, w.Result().Header.Get("link"), "service-doc")
}

func TestServiceLinksExists(t *testing.T) {
	r := NewTestRouter(t)
	r.GinEngine().Use(ServiceLinkMiddleware())
	r.GinEngine().NoRoute(Handler404())
	r.GinEngine().GET("/", func(c *gin.Context) {
		c.Header("link", `<>; rel=self`)
		AddServiceLinks(c)
		c.Data(http.StatusOK, "text/plain", []byte("Hello"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Result().StatusCode, http.StatusOK)
	assert.Contains(t, w.Result().Header.Get("link"), "service-desc")
	assert.Contains(t, w.Result().Header.Get("link"), "service-doc")
}
