package stats

import "math"

// SaturatingSum adds stat contributions without wrapping at integer limits.
// Ordinary contributions retain the same values; overflowing additions stop
// at the corresponding limit.
func SaturatingSum(values ...int) int {
	total := 0
	for _, value := range values {
		if value > 0 && total > math.MaxInt-value {
			total = math.MaxInt
		} else if value < 0 && total < math.MinInt-value {
			total = math.MinInt
		} else {
			total += value
		}
	}
	return total
}
