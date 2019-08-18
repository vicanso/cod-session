# elton-session

[![Build Status](https://img.shields.io/travis/vicanso/elton-session.svg?label=linux+build)](https://travis-ci.org/vicanso/elton-session)

Session middleware for elton, it support redis or memory store by default.

Session id store by cookie is more simple. It also support by http header or other ways for session id. 

## NewByCookie

Get session id from cookie(signed). The first time commit session, it will add cookie to http response.

```go
package main

import (
	"bytes"
	"strconv"
	"time"

	"github.com/vicanso/elton"
	session "github.com/vicanso/elton-session"
)

func main() {
	store, err := session.NewMemoryStore(10)
	if err != nil {
		panic(err)
	}
	d := elton.New()
	signedKeys := &elton.RWMutexSignedKeys{}
	signedKeys.SetKeys([]string{
		"cuttlefish",
	})
	d.SignedKeys = signedKeys

	d.Use(session.NewByCookie(session.CookieConfig{
		Store:   store,
		Signed:  true,
		Expired: 10 * time.Hour,
		GenID: func() string {
			// suggest to use uuid function
			return strconv.FormatInt(time.Now().UnixNano(), 34)
		},
		Name:     "jt",
		Path:     "/",
		MaxAge:   24 * 3600,
		HttpOnly: true,
	}))

	d.GET("/", func(c *elton.Context) (err error) {
		se := c.Get(session.Key).(*session.Session)
		views := se.GetInt("views")
		se.Set("views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return
	})

	d.ListenAndServe(":7001")
}

```

## NewByHeader

Get session id from http request header. The first time commit session, it will add a response's header to http response.

```go
package main

import (
	"bytes"
	"strconv"
	"time"

	"github.com/vicanso/elton"
	session "github.com/vicanso/elton-session"
)

func main() {
	store, err := session.NewMemoryStore(10)
	if err != nil {
		panic(err)
	}
	d := elton.New()
	signedKeys := &elton.RWMutexSignedKeys{}
	signedKeys.SetKeys([]string{
		"cuttlefish",
	})
	d.SignedKeys = signedKeys

	d.Use(session.NewByHeader(session.HeaderConfig{
		Store:   store,
		Expired: 10 * time.Hour,
		GenID: func() string {
			// suggest to use uuid function
			return strconv.FormatInt(time.Now().UnixNano(), 34)
		},
		// header's name
		Name: "jt",
	}))

	d.GET("/", func(c *elton.Context) (err error) {
		se := c.Get(session.Key).(*session.Session)
		views := se.GetInt("views")
		se.Set("views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return
	})

	d.ListenAndServe(":7001")
}
```

## NewRedisStore

Create a redis store for session.

- `client` redis.Client instance
- `opts` if client clinet is nil, will use the opts for create a redis client instance

```go
store := NewRedisStore(nil, &redis.Options{
	Addr: "localhost:6379",
})
```

## NewMemoryStore

Create a memory store for session.

- `size` max size of store

```go
store, err := NewMemoryStore(1024)
```

## NewMemoryStoreByConfig

Create a memory store for session.

- `config.Size` max size of store
- `config.SaveAs` save store sa file
- `config.Interval` flush to file's interval


```go
store, err := NewMemoryStore(MemoryStoreConfig{
	Size: 1024,
	SaveAs: "/tmp/elton-session-store",
	Interval: 60 * time.Second,
})
```