package web

type rwlocker interface {
	RLock()
	RUnlock()
	Lock()
	Unlock()
}

func lock(l rwlocker, call func()) {
	l.Lock()
	call()
	l.Unlock()
}
func rwlock(l rwlocker, call func()) {
	l.RLock()
	call()
	l.RUnlock()
}
