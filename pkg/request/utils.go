package request

import "time"

// IncrementPause возвращает функцию, которая увеличивает длительность паузы
// на заданный коэффициент, но не превышая maxPause. Возвращаемая функция
// гарантирует, что длительность паузы не будет меньше 1 секунды и не превысит maxPause.
func IncrementPause(factor float64, maxPause time.Duration) func(currentPause time.Duration) time.Duration {
	return func(currentPause time.Duration) time.Duration {
		basePause := time.Second
		newPause := time.Duration(float64(currentPause) * factor)
		if newPause < basePause {
			return basePause
		}
		if newPause > maxPause {
			return maxPause
		}
		return newPause
	}
}
