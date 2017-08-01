package route

import (
	"strconv"
	"time"
)

func GetparaStrtoInt(s string, t int) int {
	if s == "" {
		return t
	}
	m, errs := strconv.Atoi(s)
	if errs != nil {
		return t
	}
	return m
}

func Gettimeout(tu string, ts time.Duration) time.Duration {
	// if tu == "", return system level timeout
	if tu == "" {
		return ts
	}
	t, errs := strconv.Atoi(tu)
	if errs != nil {
		return ts
	}
	return time.Duration(t) * time.Second
}

func Getretry(ru string, rs int) int {
	// if ru == "", return system level retry
	if ru == "" {
		return rs
	}
	r, errs := strconv.Atoi(ru)
	if errs != nil {
		return rs
	}
	return r
}
