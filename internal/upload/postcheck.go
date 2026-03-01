package upload

// SuccessRate returns sent/total ratio in range [0, 1].
func (s Summary) SuccessRate() float64 {
	if s.Total <= 0 {
		return 0
	}
	return float64(s.Sent) / float64(s.Total)
}
