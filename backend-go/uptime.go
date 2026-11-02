package main

const uptimeWindowSeconds = 30 * 24 * 60 * 60

func (s *Store) observationSeconds() float64 {
	if s.pollInterval <= 0 {
		return 600
	}
	return s.pollInterval.Seconds()
}

func summarizeUptime(known, online float64) Uptime {
	result := Uptime{Days: 30}
	if known <= 0 {
		return result
	}
	percent := online / known * 100
	result.Percent = &percent
	result.Coverage = known / uptimeWindowSeconds * 100
	return result
}
