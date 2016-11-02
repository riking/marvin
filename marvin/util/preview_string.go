package util

func PreviewString(str string, max int) string {
	var numRunes = 0
	for index := range str {
		numRunes++
		if numRunes > max {
			return str[:index]
		}
	}
	return str
}
