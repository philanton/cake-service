package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type parsedResponse struct {
	status int
	body   []byte
}

func createRequester(t *testing.T) func(req *http.Request, err error) parsedResponse {
	return func(req *http.Request, err error) parsedResponse {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return parsedResponse{}
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		resp, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		return parsedResponse{res.StatusCode, resp}
	}
}

func prepareParams(t *testing.T, params map[string]interface{}) io.Reader {
	body, err := json.Marshal(params)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	return bytes.NewBuffer(body)
}

func newTestUserService() *UserService {
	return &UserService{
		repository: NewInMemoryUserStorage(),
	}
}

func assertStatus(t *testing.T, expected int, r parsedResponse) {
	if r.status != expected {
		t.Errorf("Unexpected response status. Expected: %d, actual: %d", expected, r.status)
	}
}

func assertBody(t *testing.T, expected string, r parsedResponse) {
	actual := string(r.body)
	if actual != expected {
		t.Errorf("Unexpected response body. Expected: %s, actual: %s", expected, actual)
	}
}

func TestUsers_Register(t *testing.T) {
	doRequest := createRequester(t)

	t.Run("email validation", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "test.mail.com",
			"password":      "somepass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "email is not valid", resp)
		ts.Close()
	})

	t.Run("password validation", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "_small_",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "password should have at least 8 symbols", resp)
		ts.Close()
	})

	t.Run("favorite cake validation", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "favorite cake should not be empty", resp)

		params["favorite_cake"] = "_cake is f@ls7"
		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "favorite cake should have only alphabetic characters", resp)

		ts.Close()
	})

	t.Run("succesful registration", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 201, resp)
		assertBody(t, "registered", resp)
		ts.Close()
	})

	t.Run("unsuccesful registration", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "user with given login is already present", resp)
		ts.Close()
	})

}

func TestUsers_JWT(t *testing.T) {
	doRequest := createRequester(t)

	t.Run("user does not exist", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(wrapJWT(j, u.JWT)))
		defer ts.Close()

		params := map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "there is no such user to get", resp)
	})

	t.Run("wrong password", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "correct_pass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(wrapJWT(j, u.JWT)))
		params = map[string]interface{}{
			"email":    "test@mail.com",
			"password": "wrong_pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "invalid login params", resp)
		ts.Close()
	})

	t.Run("unauthorized cake", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(j.jwtAuth(u.repository, getCakeHandler)))
		defer ts.Close()

		resp := doRequest(http.NewRequest(http.MethodGet, ts.URL, nil))
		assertStatus(t, 401, resp)
		assertBody(t, "unauthorized", resp)
	})

	t.Run("wrong credentials", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(j.jwtAuth(u.repository, getCakeHandler)))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		req.Header.Set("Authorization", "Bearer something strange instead of jwt")
		resp := doRequest(req, err)
		assertStatus(t, 401, resp)
		assertBody(t, "unauthorized", resp)
	})

	t.Run("authorived cake", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(wrapJWT(j, u.JWT)))
		params = map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(j.jwtAuth(u.repository, getCakeHandler)))
		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		req.Header.Set("Authorization", "Bearer "+string(resp.body))
		resp = doRequest(req, err)
		assertStatus(t, 200, resp)
		assertBody(t, "somecake", resp)
		ts.Close()
	})
}

func TestUsers_Update(t *testing.T) {
	doRequest := createRequester(t)

	t.Run("favorite cake updating", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(wrapJWT(js, us.JWT)))
		params = map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(js.jwtAuth(us.repository, us.OverwriteCake)))

		params = map[string]interface{}{
			"favorite_cake": "",
		}
		req, err := http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "favorite cake should not be empty", resp)

		params["favorite_cake"] = "some cake"
		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "favorite cake should have only alphabetic characters", resp)

		params["favorite_cake"] = "somecake"
		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "favorite cake changed", resp)
	})

	t.Run("password updating", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(wrapJWT(js, us.JWT)))
		params = map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)
		ts.Close()

		ts = httptest.NewServer(
			http.HandlerFunc(js.jwtAuth(us.repository, us.OverwritePassword)),
		)

		params = map[string]interface{}{
			"password": "pass",
		}
		req, err := http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "password should have at least 8 symbols", resp)

		params["password"] = "somepass"
		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "password changed", resp)

		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 401, resp)
		assertBody(t, "token is banned", resp)
	})

	t.Run("email updating", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(wrapJWT(js, us.JWT)))
		params = map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)
		ts.Close()

		ts = httptest.NewServer(
			http.HandlerFunc(js.jwtAuth(us.repository, us.OverwriteEmail)),
		)

		params = map[string]interface{}{
			"email": "em@il",
		}
		req, err := http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "email is not valid", resp)

		params["email"] = "test@penware.com"
		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "email changed", resp)

		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 401, resp)
		assertBody(t, "unauthorized", resp)
	})
}

func TestUsers_Admin(t *testing.T) {
	doRequest := createRequester(t)
	su_login := os.Getenv("CAKE_ADMIN_EMAIL")
	su_password := os.Getenv("CAKE_ADMIN_PASSWORD")

	t.Run("banning user", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(wrapJWT(js, us.JWT)))
		params = map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)

		params = map[string]interface{}{
			"email":    su_login,
			"password": su_password,
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		su_jwtToken := string(resp.body)

		ts = httptest.NewServer(http.HandlerFunc(js.jwtAuth(us.repository, us.History)))

		url := ts.URL + "?email=test@mail.com"
		req, err := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+su_jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "user history is clear", resp)
		ts.Close()

		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(js.jwtAuth(us.repository, us.BanUser)))
		defer ts.Close()

		params = map[string]interface{}{
			"email":  "test@mail.com",
			"reason": "some reason",
		}
		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+su_jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "user \"test@mail.com\" is banned with reason\"some reason\" by \""+su_login+"\"", resp)

		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 401, resp)
		assertBody(t, "user is banned with reason \"some reason\" by \""+su_login+"\"", resp)
	})

	t.Run("unbanning user", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewJWTService("pubkey.rsa", "privkey.rsa")
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "test@mail.com",
			"password":      "somepass",
			"favorite_cake": "somecake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(wrapJWT(js, us.JWT)))
		params = map[string]interface{}{
			"email":    "test@mail.com",
			"password": "somepass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)

		params = map[string]interface{}{
			"email":    su_login,
			"password": su_password,
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		su_jwtToken := string(resp.body)

		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(js.jwtAuth(us.repository, us.BanUser)))

		params = map[string]interface{}{
			"email":  "test@mail.com",
			"reason": "some reason",
		}
		req, err := http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+su_jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "user \"test@mail.com\" is banned with reason\"some reason\" by \""+su_login+"\"", resp)
		ts.Close()

		ts = httptest.NewServer(http.HandlerFunc(js.jwtAuth(us.repository, us.UnbanUser)))
		defer ts.Close()

		params = map[string]interface{}{
			"email": "test@mail.com",
		}
		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+su_jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "user \"test@mail.com\" is unbanned by \""+su_login+"\"", resp)

		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "not enough privileges", resp)
	})
}
