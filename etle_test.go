package main

import (
	"errors"
	"testing"
	"time"
)

func TestTokenExpired(t *testing.T) {
	now := time.Now().Unix()

	mk := func(exp int64) string {
		payload := `{"exp":` + itoa(exp) + `}`
		return "aaa." + b64url(payload) + ".sss"
	}

	if !TokenExpired("") {
		t.Error("token kosong harus dianggap expired")
	}
	if TokenExpired("not-a-jwt") {
		t.Error("token non-JWT tidak boleh dipaksa expired")
	}
	if TokenExpired(mk(now+3600)) {
		t.Error("token belum expired tidak boleh dianggap expired")
	}
	if !TokenExpired(mk(now-60)) {
		t.Error("token sudah lewat exp harus dianggap expired")
	}
	if !TokenExpired(mk(now)) {
		t.Error("token tepat di exp harus dianggap expired")
	}
}

func TestIsTokenExpired(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&APIError{401, "TokenExpiredError jwt expired"}, true},
		{&APIError{403, `{"name":"TokenExpiredError","message":"jwt expired"}`}, true},
		{&APIError{403, "token expired atau tidak valid"}, true},
		{&APIError{403, "forbidden: anda tidak punya akses"}, false},
		{&APIError{500, "jwt expired"}, false},
		{errors.New("network down"), false},
	}
	for _, c := range cases {
		if got := IsTokenExpired(c.err); got != c.want {
			t.Errorf("IsTokenExpired(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [24]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func b64url(s string) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	out := ""
	for len(s) >= 3 {
		c := (uint32(s[0])<<16 | uint32(s[1])<<8 | uint32(s[2]))
		out += string(chars[(c>>18)&63]) + string(chars[(c>>12)&63]) + string(chars[(c>>6)&63]) + string(chars[c&63])
		s = s[3:]
	}
	switch len(s) {
	case 1:
		c := uint32(s[0]) << 16
		out += string(chars[(c>>18)&63]) + string(chars[(c>>12)&63])
	case 2:
		c := uint32(s[0])<<16 | uint32(s[1])<<8
		out += string(chars[(c>>18)&63]) + string(chars[(c>>12)&63]) + string(chars[(c>>6)&63])
	}
	return out
}