// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//nolint:forbidigo
package integration

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/testlogger"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers"
	user_service "code.gitea.io/gitea/services/user"
	"code.gitea.io/gitea/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	goth_gitlab "github.com/markbates/goth/providers/gitlab"
	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

var testWebRoutes *web.Route

type NilResponseRecorder struct {
	httptest.ResponseRecorder
	Length int
}

func (n *NilResponseRecorder) Write(b []byte) (int, error) {
	n.Length += len(b)
	return len(b), nil
}

// NewRecorder returns an initialized ResponseRecorder.
func NewNilResponseRecorder() *NilResponseRecorder {
	return &NilResponseRecorder{
		ResponseRecorder: *httptest.NewRecorder(),
	}
}

type NilResponseHashSumRecorder struct {
	httptest.ResponseRecorder
	Hash   hash.Hash
	Length int
}

func (n *NilResponseHashSumRecorder) Write(b []byte) (int, error) {
	_, _ = n.Hash.Write(b)
	n.Length += len(b)
	return len(b), nil
}

// NewRecorder returns an initialized ResponseRecorder.
func NewNilResponseHashSumRecorder() *NilResponseHashSumRecorder {
	return &NilResponseHashSumRecorder{
		Hash:             fnv.New32(),
		ResponseRecorder: *httptest.NewRecorder(),
	}
}

func TestMain(m *testing.M) {
	defer log.GetManager().Close()

	managerCtx, cancel := context.WithCancel(context.Background())
	graceful.InitManager(managerCtx)
	defer cancel()

	tests.InitTest(true)
	testWebRoutes = routers.NormalRoutes()

	// integration test settings...
	if setting.CfgProvider != nil {
		testingCfg := setting.CfgProvider.Section("integration-tests")
		testlogger.SlowTest = testingCfg.Key("SLOW_TEST").MustDuration(testlogger.SlowTest)
		testlogger.SlowFlush = testingCfg.Key("SLOW_FLUSH").MustDuration(testlogger.SlowFlush)
	}

	if os.Getenv("GITEA_SLOW_TEST_TIME") != "" {
		duration, err := time.ParseDuration(os.Getenv("GITEA_SLOW_TEST_TIME"))
		if err == nil {
			testlogger.SlowTest = duration
		}
	}

	if os.Getenv("GITEA_SLOW_FLUSH_TIME") != "" {
		duration, err := time.ParseDuration(os.Getenv("GITEA_SLOW_FLUSH_TIME"))
		if err == nil {
			testlogger.SlowFlush = duration
		}
	}

	os.Unsetenv("GIT_AUTHOR_NAME")
	os.Unsetenv("GIT_AUTHOR_EMAIL")
	os.Unsetenv("GIT_AUTHOR_DATE")
	os.Unsetenv("GIT_COMMITTER_NAME")
	os.Unsetenv("GIT_COMMITTER_EMAIL")
	os.Unsetenv("GIT_COMMITTER_DATE")

	err := unittest.InitFixtures(
		unittest.FixturesOptions{
			Dir: filepath.Join(filepath.Dir(setting.AppPath), "models/fixtures/"),
		},
	)
	if err != nil {
		fmt.Printf("Error initializing test database: %v\n", err)
		os.Exit(1)
	}

	// FIXME: the console logger is deleted by mistake, so if there is any `log.Fatal`, developers won't see any error message.
	// Instead, "No tests were found",  last nonsense log is "According to the configuration, subsequent logs will not be printed to the console"
	exitCode := m.Run()

	testlogger.WriterCloser.Reset()

	if err = util.RemoveAll(setting.Indexer.IssuePath); err != nil {
		fmt.Printf("util.RemoveAll: %v\n", err)
		os.Exit(1)
	}
	if err = util.RemoveAll(setting.Indexer.RepoPath); err != nil {
		fmt.Printf("Unable to remove repo indexer: %v\n", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

type TestSession struct {
	jar http.CookieJar
}

func (s *TestSession) GetCookie(name string) *http.Cookie {
	baseURL, err := url.Parse(setting.AppURL)
	if err != nil {
		return nil
	}

	for _, c := range s.jar.Cookies(baseURL) {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (s *TestSession) MakeRequest(t testing.TB, rw *RequestWrapper, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := rw.Request
	baseURL, err := url.Parse(setting.AppURL)
	assert.NoError(t, err)
	for _, c := range s.jar.Cookies(baseURL) {
		req.AddCookie(c)
	}
	resp := MakeRequest(t, rw, expectedStatus)

	ch := http.Header{}
	ch.Add("Cookie", strings.Join(resp.Header()["Set-Cookie"], ";"))
	cr := http.Request{Header: ch}
	s.jar.SetCookies(baseURL, cr.Cookies())

	return resp
}

func (s *TestSession) MakeRequestNilResponseRecorder(t testing.TB, rw *RequestWrapper, expectedStatus int) *NilResponseRecorder {
	t.Helper()
	req := rw.Request
	baseURL, err := url.Parse(setting.AppURL)
	assert.NoError(t, err)
	for _, c := range s.jar.Cookies(baseURL) {
		req.AddCookie(c)
	}
	resp := MakeRequestNilResponseRecorder(t, rw, expectedStatus)

	ch := http.Header{}
	ch.Add("Cookie", strings.Join(resp.Header()["Set-Cookie"], ";"))
	cr := http.Request{Header: ch}
	s.jar.SetCookies(baseURL, cr.Cookies())

	return resp
}

func (s *TestSession) MakeRequestNilResponseHashSumRecorder(t testing.TB, rw *RequestWrapper, expectedStatus int) *NilResponseHashSumRecorder {
	t.Helper()
	req := rw.Request
	baseURL, err := url.Parse(setting.AppURL)
	assert.NoError(t, err)
	for _, c := range s.jar.Cookies(baseURL) {
		req.AddCookie(c)
	}
	resp := MakeRequestNilResponseHashSumRecorder(t, rw, expectedStatus)

	ch := http.Header{}
	ch.Add("Cookie", strings.Join(resp.Header()["Set-Cookie"], ";"))
	cr := http.Request{Header: ch}
	s.jar.SetCookies(baseURL, cr.Cookies())

	return resp
}

const userPassword = "password"

func emptyTestSession(t testing.TB) *TestSession {
	t.Helper()
	jar, err := cookiejar.New(nil)
	assert.NoError(t, err)

	return &TestSession{jar: jar}
}

func getUserToken(t testing.TB, userName string, scope ...auth.AccessTokenScope) string {
	return getTokenForLoggedInUser(t, loginUser(t, userName), scope...)
}

func mockCompleteUserAuth(mock func(res http.ResponseWriter, req *http.Request) (goth.User, error)) func() {
	old := gothic.CompleteUserAuth
	gothic.CompleteUserAuth = mock
	return func() {
		gothic.CompleteUserAuth = old
	}
}

func addAuthSource(t *testing.T, payload map[string]string) *auth.Source {
	session := loginUser(t, "user1")
	payload["_csrf"] = GetCSRF(t, session, "/admin/auths/new")
	req := NewRequestWithValues(t, "POST", "/admin/auths/new", payload)
	session.MakeRequest(t, req, http.StatusSeeOther)
	source, err := auth.GetSourceByName(context.Background(), payload["name"])
	assert.NoError(t, err)
	return source
}

func authSourcePayloadOAuth2(name string) map[string]string {
	return map[string]string{
		"type":      fmt.Sprintf("%d", auth.OAuth2),
		"name":      name,
		"is_active": "on",
	}
}

func authSourcePayloadGitLab(name string) map[string]string {
	payload := authSourcePayloadOAuth2(name)
	payload["oauth2_provider"] = "gitlab"
	return payload
}

func authSourcePayloadGitLabCustom(name string) map[string]string {
	payload := authSourcePayloadGitLab(name)
	payload["oauth2_use_custom_url"] = "on"
	payload["oauth2_auth_url"] = goth_gitlab.AuthURL
	payload["oauth2_token_url"] = goth_gitlab.TokenURL
	payload["oauth2_profile_url"] = goth_gitlab.ProfileURL
	return payload
}

func createUser(ctx context.Context, t testing.TB, user *user_model.User) func() {
	user.MustChangePassword = false
	user.LowerName = strings.ToLower(user.Name)

	assert.NoError(t, db.Insert(ctx, user))

	if len(user.Email) > 0 {
		changePrimaryEmail := true
		assert.NoError(t, user_model.UpdateUser(ctx, user, changePrimaryEmail))
	}

	return func() {
		assert.NoError(t, user_service.DeleteUser(ctx, user, true))
	}
}

func loginUser(t testing.TB, userName string) *TestSession {
	t.Helper()

	return loginUserWithPassword(t, userName, userPassword)
}

func loginUserWithPassword(t testing.TB, userName, password string) *TestSession {
	t.Helper()

	return loginUserWithPasswordRemember(t, userName, password, false)
}

func loginUserWithPasswordRemember(t testing.TB, userName, password string, rememberMe bool) *TestSession {
	t.Helper()
	req := NewRequest(t, "GET", "/user/login")
	resp := MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)
	req = NewRequestWithValues(t, "POST", "/user/login", map[string]string{
		"_csrf":     doc.GetCSRF(),
		"user_name": userName,
		"password":  password,
		"remember":  strconv.FormatBool(rememberMe),
	})
	resp = MakeRequest(t, req, http.StatusSeeOther)

	ch := http.Header{}
	ch.Add("Cookie", strings.Join(resp.Header()["Set-Cookie"], ";"))
	cr := http.Request{Header: ch}

	session := emptyTestSession(t)

	baseURL, err := url.Parse(setting.AppURL)
	assert.NoError(t, err)
	session.jar.SetCookies(baseURL, cr.Cookies())

	return session
}

// token has to be unique this counter take care of
var tokenCounter int64

// getTokenForLoggedInUser returns a token for a logged in user.
// The scope is an optional list of snake_case strings like the frontend form fields,
// but without the "scope_" prefix.
func getTokenForLoggedInUser(t testing.TB, session *TestSession, scopes ...auth.AccessTokenScope) string {
	t.Helper()
	var token string
	req := NewRequest(t, "GET", "/user/settings/applications")
	resp := session.MakeRequest(t, req, http.StatusOK)
	var csrf string
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name != "_csrf" {
			continue
		}
		csrf = cookie.Value
		break
	}
	if csrf == "" {
		doc := NewHTMLParser(t, resp.Body)
		csrf = doc.GetCSRF()
	}
	assert.NotEmpty(t, csrf)
	urlValues := url.Values{}
	urlValues.Add("_csrf", csrf)
	urlValues.Add("name", fmt.Sprintf("api-testing-token-%d", atomic.AddInt64(&tokenCounter, 1)))
	for _, scope := range scopes {
		urlValues.Add("scope", string(scope))
	}
	req = NewRequestWithURLValues(t, "POST", "/user/settings/applications", urlValues)
	resp = session.MakeRequest(t, req, http.StatusSeeOther)

	// Log the flash values on failure
	if !assert.Equal(t, resp.Result().Header["Location"], []string{"/user/settings/applications"}) {
		for _, cookie := range resp.Result().Cookies() {
			if cookie.Name != gitea_context.CookieNameFlash {
				continue
			}
			flash, _ := url.ParseQuery(cookie.Value)
			for key, value := range flash {
				t.Logf("Flash %q: %q", key, value)
			}
		}
	}

	req = NewRequest(t, "GET", "/user/settings/applications")
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	token = htmlDoc.doc.Find(".ui.info p").Text()
	assert.NotEmpty(t, token)
	return token
}

type RequestWrapper struct {
	*http.Request
}

func (req *RequestWrapper) AddBasicAuth(username string) *RequestWrapper {
	req.Request.SetBasicAuth(username, userPassword)
	return req
}

func (req *RequestWrapper) AddTokenAuth(token string) *RequestWrapper {
	if token == "" {
		return req
	}
	if !strings.HasPrefix(token, "Bearer ") {
		token = "Bearer " + token
	}
	req.Request.Header.Set("Authorization", token)
	return req
}

func (req *RequestWrapper) SetHeader(name, value string) *RequestWrapper {
	req.Request.Header.Set(name, value)
	return req
}

func NewRequest(t testing.TB, method, urlStr string) *RequestWrapper {
	t.Helper()
	return NewRequestWithBody(t, method, urlStr, nil)
}

func NewRequestf(t testing.TB, method, urlFormat string, args ...any) *RequestWrapper {
	t.Helper()
	return NewRequest(t, method, fmt.Sprintf(urlFormat, args...))
}

func NewRequestWithValues(t testing.TB, method, urlStr string, values map[string]string) *RequestWrapper {
	t.Helper()
	urlValues := url.Values{}
	for key, value := range values {
		urlValues[key] = []string{value}
	}
	return NewRequestWithURLValues(t, method, urlStr, urlValues)
}

func NewRequestWithURLValues(t testing.TB, method, urlStr string, urlValues url.Values) *RequestWrapper {
	t.Helper()
	return NewRequestWithBody(t, method, urlStr, bytes.NewBufferString(urlValues.Encode())).
		SetHeader("Content-Type", "application/x-www-form-urlencoded")
}

func NewRequestWithJSON(t testing.TB, method, urlStr string, v any) *RequestWrapper {
	t.Helper()

	jsonBytes, err := json.Marshal(v)
	assert.NoError(t, err)
	return NewRequestWithBody(t, method, urlStr, bytes.NewBuffer(jsonBytes)).
		SetHeader("Content-Type", "application/json")
}

func NewRequestWithBody(t testing.TB, method, urlStr string, body io.Reader) *RequestWrapper {
	t.Helper()
	if !strings.HasPrefix(urlStr, "http") && !strings.HasPrefix(urlStr, "/") {
		urlStr = "/" + urlStr
	}
	req, err := http.NewRequest(method, urlStr, body)
	assert.NoError(t, err)
	req.RequestURI = urlStr

	return &RequestWrapper{req}
}

const NoExpectedStatus = -1

func MakeRequest(t testing.TB, rw *RequestWrapper, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := rw.Request
	recorder := httptest.NewRecorder()
	if req.RemoteAddr == "" {
		req.RemoteAddr = "test-mock:12345"
	}
	testWebRoutes.ServeHTTP(recorder, req)
	if expectedStatus != NoExpectedStatus {
		if !assert.EqualValues(t, expectedStatus, recorder.Code, "Request: %s %s", req.Method, req.URL.String()) {
			logUnexpectedResponse(t, recorder)
		}
	}
	return recorder
}

func MakeRequestNilResponseRecorder(t testing.TB, rw *RequestWrapper, expectedStatus int) *NilResponseRecorder {
	t.Helper()
	req := rw.Request
	recorder := NewNilResponseRecorder()
	testWebRoutes.ServeHTTP(recorder, req)
	if expectedStatus != NoExpectedStatus {
		if !assert.EqualValues(t, expectedStatus, recorder.Code,
			"Request: %s %s", req.Method, req.URL.String()) {
			logUnexpectedResponse(t, &recorder.ResponseRecorder)
		}
	}
	return recorder
}

func MakeRequestNilResponseHashSumRecorder(t testing.TB, rw *RequestWrapper, expectedStatus int) *NilResponseHashSumRecorder {
	t.Helper()
	req := rw.Request
	recorder := NewNilResponseHashSumRecorder()
	testWebRoutes.ServeHTTP(recorder, req)
	if expectedStatus != NoExpectedStatus {
		if !assert.EqualValues(t, expectedStatus, recorder.Code,
			"Request: %s %s", req.Method, req.URL.String()) {
			logUnexpectedResponse(t, &recorder.ResponseRecorder)
		}
	}
	return recorder
}

// logUnexpectedResponse logs the contents of an unexpected response.
func logUnexpectedResponse(t testing.TB, recorder *httptest.ResponseRecorder) {
	t.Helper()
	respBytes := recorder.Body.Bytes()
	if len(respBytes) == 0 {
		return
	} else if len(respBytes) < 500 {
		// if body is short, just log the whole thing
		t.Log("Response: ", string(respBytes))
		return
	}
	t.Log("Response length: ", len(respBytes))

	// log the "flash" error message, if one exists
	// we must create a new buffer, so that we don't "use up" resp.Body
	htmlDoc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(respBytes))
	if err != nil {
		return // probably a non-HTML response
	}
	errMsg := htmlDoc.Find(".ui.negative.message").Text()
	if len(errMsg) > 0 {
		t.Log("A flash error message was found:", errMsg)
	}
}

func DecodeJSON(t testing.TB, resp *httptest.ResponseRecorder, v any) {
	t.Helper()

	decoder := json.NewDecoder(resp.Body)
	assert.NoError(t, decoder.Decode(v))
}

func VerifyJSONSchema(t testing.TB, resp *httptest.ResponseRecorder, schemaFile string) {
	t.Helper()

	schemaFilePath := filepath.Join(filepath.Dir(setting.AppPath), "tests", "integration", "schemas", schemaFile)
	_, schemaFileErr := os.Stat(schemaFilePath)
	assert.Nil(t, schemaFileErr)

	schema, schemaFileReadErr := os.ReadFile(schemaFilePath)
	assert.Nil(t, schemaFileReadErr)
	assert.True(t, len(schema) > 0)

	nodeinfoSchema := gojsonschema.NewStringLoader(string(schema))
	nodeinfoString := gojsonschema.NewStringLoader(resp.Body.String())
	result, schemaValidationErr := gojsonschema.Validate(nodeinfoSchema, nodeinfoString)
	assert.Nil(t, schemaValidationErr)
	assert.Empty(t, result.Errors())
	assert.True(t, result.Valid())
}

func GetCSRF(t testing.TB, session *TestSession, urlStr string) string {
	t.Helper()
	req := NewRequest(t, "GET", urlStr)
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	return doc.GetCSRF()
}
