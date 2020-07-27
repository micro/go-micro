package qson

func Fuzz(data []byte) int {
	_, err := ToJSON(string(data))
	if err != nil {
		return 0
	}
	return 1
}
