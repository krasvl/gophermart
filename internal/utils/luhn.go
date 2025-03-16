package utils

func IsValidLuhn(number string) bool {
	var sum int
	var double bool
	for i := len(number) - 1; i >= 0; i-- {
		n := int(number[i] - '0')
		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}
	return sum%10 == 0
}
