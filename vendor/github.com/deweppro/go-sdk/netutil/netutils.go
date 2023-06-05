package netutil

import (
	"net"
	"reflect"
	"strings"

	"github.com/deweppro/go-sdk/errors"
)

var (
	ErrResolveTCPAddress = errors.New("resolve tcp address")
)

func RandomPort(host string) (string, error) {
	host = strings.Join([]string{host, "0"}, ":")
	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return host, errors.Wrap(err, ErrResolveTCPAddress)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return host, errors.Wrap(err, ErrResolveTCPAddress)
	}
	v := l.Addr().String()
	if err = l.Close(); err != nil {
		return host, errors.Wrap(err, ErrResolveTCPAddress)
	}
	return v, nil
}

func FileDescriptor(c net.Conn) int {
	fd := reflect.Indirect(reflect.ValueOf(c)).FieldByName("fd")
	pfd := reflect.Indirect(fd).FieldByName("pfd")
	return int(pfd.FieldByName("Sysfd").Int())
}

func CheckHostPort(addr string) string {
	hp := strings.Split(addr, ":")
	if len(hp) != 2 {
		tmp := make([]string, 2)
		copy(hp, tmp)
		hp = tmp
	}
	if len(hp[0]) == 0 {
		hp[0] = "0.0.0.0"
	}
	if len(hp[1]) == 0 {
		if v, err := RandomPort(hp[0]); err == nil {
			hp[1] = v
		} else {
			hp[1] = "8080"
		}
	}
	return strings.Join(hp, ":")
}
