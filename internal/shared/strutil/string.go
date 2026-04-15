package strutil

// OnlyNumbers removes all non-numeric characters from a string.
func OnlyNumbers(s string) string {
	res := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			res = append(res, s[i])
		}
	}
	return string(res)
}
