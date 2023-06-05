package signature

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"sync"
)

var _ Signature = (*_sig)(nil)

type (
	_sig struct {
		id       string
		hashFunc hash.Hash
		alg      string
		lock     sync.Mutex
	}

	//Signature interface
	Signature interface {
		ID() string
		Algorithm() string
		Create(b []byte) []byte
		CreateString(b []byte) string
		Validate(b []byte, ex string) bool
	}
)

// NewSHA256 create sign sha256
func NewSHA256(id, secret string) Signature {
	return NewCustomSignature(id, secret, "hmac-sha256", sha256.New)
}

// NewMD5 create sign md5
func NewMD5(id, secret string) Signature {
	return NewCustomSignature(id, secret, "hmac-md5", md5.New)
}

// NewSHA512 create sign sha512
func NewSHA512(id, secret string) Signature {
	return NewCustomSignature(id, secret, "hmac-sha512", sha512.New)
}

// NewCustomSignature create sign with custom hash function
func NewCustomSignature(id, secret, alg string, h func() hash.Hash) Signature {
	return &_sig{
		id:       id,
		alg:      alg,
		hashFunc: hmac.New(h, []byte(secret)),
	}
}

// ID signature
func (s *_sig) ID() string {
	return s.id
}

// Algorithm getter
func (s *_sig) Algorithm() string {
	return s.alg
}

// Create getting hash as bytes
func (s *_sig) Create(b []byte) []byte {
	s.lock.Lock()
	defer func() {
		s.hashFunc.Reset()
		s.lock.Unlock()
	}()
	s.hashFunc.Write(b)
	return s.hashFunc.Sum(nil)
}

// CreateString getting hash as string
func (s *_sig) CreateString(b []byte) string {
	return hex.EncodeToString(s.Create(b))
}

// Validate signature
func (s *_sig) Validate(b []byte, ex string) bool {
	v, err := hex.DecodeString(ex)
	if err != nil {
		return false
	}
	return hmac.Equal(s.Create(b), v)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// _store storage
type (
	_store struct {
		list map[string]Signature
		lock sync.RWMutex
	}

	Storage interface {
		Add(s Signature)
		Get(id string) Signature
		Count() int
		Del(id string)
		Flush()
	}
)

// NewStorage init storage
func NewStorage() Storage {
	return &_store{
		list: make(map[string]Signature),
	}
}

// Add adding to storage
func (ss *_store) Add(s Signature) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.list[s.ID()] = s
}

// Get getting to storage
func (ss *_store) Get(id string) Signature {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	if s, ok := ss.list[id]; ok {
		return s
	}
	return nil
}

// Count count sign in storage
func (ss *_store) Count() int {
	ss.lock.RLock()
	defer ss.lock.RUnlock()
	l := len(ss.list)
	return l
}

// Del deleting from storage
func (ss *_store) Del(id string) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	delete(ss.list, id)
}

// Flush removing all from storage
func (ss *_store) Flush() {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	for k := range ss.list {
		delete(ss.list, k)
	}
}
