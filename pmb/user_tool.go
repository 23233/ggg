package pmb

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/23233/ggg/ut"
)

func passwordSalt(rawPassword string) (ps string, salt string) {
	salt = ut.RandomStr(4)
	m5 := md5.New()
	m5.Write([]byte(rawPassword))
	m5.Write([]byte(salt))
	st := m5.Sum(nil)
	ps = hex.EncodeToString(st)
	return ps, salt
}

func validPassword(password, salt, m5 string) bool {
	r := md5.New()
	r.Write([]byte(password))
	r.Write([]byte(salt))
	st := r.Sum(nil)
	ps := hex.EncodeToString(st)
	return ps == m5
}
