package toolkit

import "crypto/rand"

const randomStringSrc = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type Tools struct{}

func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSrc)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x,y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]

	}
	return string(s)
}