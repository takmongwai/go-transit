package main

import "errors"
import "sync"

type AccessUser struct {
	ID       string `json:"id"`
	Password string `json:"password"`
	Active   bool   `json:"active"`
}

type AccessUserMap struct {
	sync.RWMutex
	m map[string]*AccessUser
}

func NewAccessUserMap() *AccessUserMap {
	return &AccessUserMap{
		m: make(map[string]*AccessUser),
	}
}

func (aum *AccessUserMap) Put(id string, au *AccessUser) {
	aum.Lock()
	defer aum.Unlock()
	aum.m[id] = au
}

func (aum *AccessUserMap) Get(id string) *AccessUser {
	aum.RLock()
	defer aum.RUnlock()
	if v, ok := aum.m[id]; ok {
		return v
	}
	return nil
}

// CheckByIDAndPassword 根据用户id和密码检查用户
func (aum *AccessUserMap) CheckByIDAndPassword(id, pass string) error {
	v := aum.Get(id)
	switch {
	case v == nil:
		return errors.New("unknown user")
	case v.Password != pass:
		return errors.New("error password")
	case !v.Active:
		return errors.New("Inactive user")
	default:
		return nil
	}
}
