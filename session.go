// MIT License

// Copyright (c) 2020 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package session

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/spf13/cast"
	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

const (
	// CreatedAt the created time for session
	CreatedAt = "_createdAt"
	// UpdatedAt the updated time for session
	UpdatedAt = "_updatedAt"
	// ErrCategory session error category
	ErrCategory = "elton-session"
	// Key session key
	Key = "_session"
)

var (
	// ErrDuplicateCommit duplicate commit
	ErrDuplicateCommit = createError("duplicate commit")
	// ErrIDNil session id is nil
	ErrIDNil = createError("session id is nil")
	// ErrIsReadonly session is readonly
	ErrIsReadonly = createError("session is readonly")
)

type (
	// M alias
	M map[string]interface{}
	// Config session middleware config
	Config struct {
		// LazyFetch if set true, the fetch function isn't called when initialization
		LazyFetch bool
		// Store session store
		Store Store
		// Skipper skipper
		Skipper elton.Skipper
		// Expired session store's max age
		Expired time.Duration
		// GenID generate uid
		GenID func() string

		Get func(c *elton.Context) (string, error)
		Set func(c *elton.Context, id string) error
	}
	// CookieConfig session cookie config
	CookieConfig struct {
		// Store session store
		Store Store
		// Skipper skipper
		Skipper elton.Skipper
		// Expired session store's max age
		Expired time.Duration
		// GenID generate uid
		GenID func() string

		// Signed signed cookie
		Signed bool
		// Cookie cookie config
		Name     string
		Path     string
		Domain   string
		MaxAge   int
		Secure   bool
		HttpOnly bool
	}
	// HeaderConfig session header config
	HeaderConfig struct {
		// Store session store
		Store Store
		// Skipper skipper
		Skipper elton.Skipper
		// Expired session store's max age
		Expired time.Duration
		// GenID generate uid
		GenID func() string

		// Name header's name
		Name string
	}
	// Session session struct
	Session struct {
		// Store session store
		Store Store
		// ID the session id
		ID string
		// the data fetch from session
		data M
		// the data has been fetched
		fetched bool
		// the data has been modified
		modified bool
		// the session has been committed
		committed bool
		// the session is readonly
		readonly bool
		// ignore the modify(not sync data to store)
		ignoreModified bool
	}
	// Store session store
	Store interface {
		// Get get the session data
		Get(string) ([]byte, error)
		// Set set the session data
		Set(string, []byte, time.Duration) error
		// Destroy remove the session data
		Destroy(string) error
	}
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// generateID gen id
func generateID() string {
	b := make([]rune, 24)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func createError(message string) *hes.Error {
	return &hes.Error{
		Message:    message,
		Category:   ErrCategory,
		StatusCode: http.StatusInternalServerError,
		Exception:  true,
	}
}

func wrapError(err error) *hes.Error {
	he, ok := err.(*hes.Error)
	if !ok {
		he = hes.NewWithError(err)
		he.StatusCode = http.StatusInternalServerError
		he.Category = ErrCategory
	}
	he.Exception = true
	return he
}

func getInitMap() M {
	m := make(M)
	m[CreatedAt] = time.Now().Format(time.RFC3339)
	return m
}

func (s *Session) fetch() (err error) {
	if s.fetched {
		return
	}
	store := s.Store
	var buf []byte
	if s.ID != "" {
		buf, err = store.Get(s.ID)
		if err != nil {
			return
		}
	}
	m := make(M)
	if len(buf) == 0 {
		m = getInitMap()
	} else {
		err = json.Unmarshal(buf, &m)
	}
	if err != nil {
		return
	}
	s.fetched = true
	s.data = m
	return
}

// Fetch fetch the session data from store
func (s *Session) Fetch() (m M, err error) {
	if s.fetched {
		m = s.data
		return
	}
	err = s.fetch()
	if err != nil {
		return
	}
	m = s.data
	return
}

// Destroy remove the data from store and reset session data
func (s *Session) Destroy() (err error) {
	if s.ID == "" {
		return
	}
	store := s.Store
	m := getInitMap()
	s.data = m
	err = store.Destroy(s.ID)
	if err != nil {
		return
	}
	s.ID = ""
	return
}

func (s *Session) updatedAt() {
	s.data[UpdatedAt] = time.Now().Format(time.RFC3339)
	s.modified = true
}

// Set set data to session
func (s *Session) Set(key string, value interface{}) (err error) {
	if s.readonly {
		return ErrIsReadonly
	}
	if key == "" {
		return
	}
	if !s.fetched {
		err = s.fetch()
		if err != nil {
			return
		}
	}
	if value == nil {
		delete(s.data, key)
	} else {
		s.data[key] = value
	}
	s.updatedAt()
	return
}

// SetMap set map data to session
func (s *Session) SetMap(value map[string]interface{}) (err error) {
	if s.readonly {
		return ErrIsReadonly
	}
	if value == nil {
		return
	}
	if !s.fetched {
		err = s.fetch()
		if err != nil {
			return
		}
	}
	for k, v := range value {
		if v == nil {
			delete(s.data, k)
			continue
		}
		s.data[k] = v
	}

	s.updatedAt()
	return
}

// Readonly
func (s *Session) Readonly() bool {
	return s.readonly
}

// EnableReadonly enable session readonly
func (s *Session) EnableReadonly() {
	s.readonly = true
}

// Refresh refresh session (update updatedAt)
func (s *Session) Refresh() (err error) {
	if !s.fetched {
		err = s.fetch()
		if err != nil {
			return
		}
	}
	s.updatedAt()
	return
}

// Get get data from session's data
func (s *Session) Get(key string) interface{} {
	if !s.fetched {
		return nil
	}
	return s.data[key]
}

// GetBool get bool data from session's data
func (s *Session) GetBool(key string) bool {
	return cast.ToBool(s.Get(key))
}

// GetString get string data from session's data
func (s *Session) GetString(key string) string {
	return cast.ToString(s.Get(key))
}

// GetInt get int data from session's data
func (s *Session) GetInt(key string) int {
	return cast.ToInt(s.Get(key))
}

// GetFloat64 get float64 data from session's data
func (s *Session) GetFloat64(key string) float64 {
	return cast.ToFloat64(s.Get(key))
}

// GetStringSlice get string slice data from session's data
func (s *Session) GetStringSlice(key string) []string {
	return cast.ToStringSlice(s.Get(key))
}

// GetCreatedAt get the created at of session
func (s *Session) GetCreatedAt() string {
	return cast.ToString(s.Get(CreatedAt))
}

// GetUpdatedAt get the updated at of session
func (s *Session) GetUpdatedAt() string {
	return cast.ToString(s.Get(UpdatedAt))
}

// GetData get the session's data
func (s *Session) GetData() M {
	return s.data
}

// EnableIgnoreModified changes the ignore modified to true
func (s *Session) EnableIgnoreModified() {
	s.ignoreModified = true
}

// Commit sync the session to store
func (s *Session) Commit(ttl time.Duration) (err error) {
	if !s.modified || s.ignoreModified {
		return
	}
	if s.committed {
		err = ErrDuplicateCommit
		return
	}
	// 如果session id为空，则生成新的session id
	if s.ID == "" {
		err = ErrIDNil
		return
	}

	buf, err := json.Marshal(s.data)
	if err != nil {
		return
	}

	err = s.Store.Set(s.ID, buf, ttl)
	if err != nil {
		return
	}
	s.committed = true
	return
}

// New create a new session middleware
func New(config Config) elton.Handler {
	store := config.Store
	getID := config.Get
	setID := config.Set
	genID := config.GenID
	expired := config.Expired
	if store == nil ||
		getID == nil ||
		setID == nil ||
		genID == nil ||
		expired == 0 {
		panic("require store, get function, set function and expired")
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	return func(c *elton.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		_, exists := c.Get(Key)
		if exists {
			return c.Next()
		}
		s := &Session{
			Store: store,
		}
		id, err := getID(c)
		if err != nil {
			err = wrapError(err)
			return
		}
		if id != "" {
			s.ID = id
		}
		// 拉取session（默认都拉取，未做动态拉取）
		if !config.LazyFetch {
			err = s.fetch()
			if err != nil {
				err = wrapError(err)
				return
			}
		}

		// session 保存至context中
		c.Set(Key, s)
		// 其它中间件的异常，不需要wrap
		err = c.Next()
		if err != nil {
			return
		}
		if s.modified {
			// 如果session 有修改而且未生成session id
			if s.ID == "" {
				uid := genID()
				err = setID(c, uid)
				if err != nil {
					err = wrapError(err)
					return
				}
				s.ID = uid
			}
			// 提交session 数据
			err = s.Commit(expired)
			if err != nil {
				err = wrapError(err)
			}
		}
		return
	}
}

// NewByCookie create a session by cookie, which get session id from cookie
func NewByCookie(config CookieConfig) elton.Handler {
	if config.Name == "" {
		panic("require cookie's name")
	}
	getID := func(c *elton.Context) (string, error) {
		getCookie := c.Cookie
		if config.Signed {
			getCookie = c.SignedCookie
		}
		// cookie只会因为获取不到而报错，因此忽略
		cookie, _ := getCookie(config.Name)
		if cookie == nil {
			return "", nil
		}
		return cookie.Value, nil
	}
	setID := func(c *elton.Context, id string) (err error) {
		setCookie := c.AddCookie
		if config.Signed {
			setCookie = c.AddSignedCookie
		}

		// 设置cookie
		setCookie(&http.Cookie{
			Name:     config.Name,
			Value:    id,
			Path:     config.Path,
			Domain:   config.Domain,
			MaxAge:   config.MaxAge,
			Secure:   config.Secure,
			HttpOnly: config.HttpOnly,
		})
		return
	}

	return New(Config{
		Store:   config.Store,
		Get:     getID,
		Set:     setID,
		GenID:   config.GenID,
		Expired: config.Expired,
	})
}

// NewByHeader create a session by header, which get session id from request header
func NewByHeader(config HeaderConfig) elton.Handler {
	if config.Name == "" {
		panic("require header's name")
	}
	getID := func(c *elton.Context) (string, error) {
		// get session id from request header
		id := c.GetRequestHeader(config.Name)
		return id, nil
	}
	setID := func(c *elton.Context, id string) (err error) {
		// set session id to response id
		c.SetHeader(config.Name, id)
		return
	}
	return New(Config{
		Store:   config.Store,
		Get:     getID,
		Set:     setID,
		GenID:   config.GenID,
		Expired: config.Expired,
	})
}
