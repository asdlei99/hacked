package movie

import "time"

const fractionDivisor = 0x10000

// Timestamp represents a point in time using fixed resolution.
type Timestamp struct {
	Second   uint8
	Fraction uint16
}

// TimestampFromSeconds creates a timestamp instance from given floating point value.
func TimestampFromSeconds(value float32) Timestamp {
	second := uint8(value)
	return Timestamp{
		Second:   second,
		Fraction: uint16((value - float32(second)) * fractionDivisor),
	}
}

// ToDuration returns the equivalent duration for this timestamp.
func (ts Timestamp) ToDuration() time.Duration {
	return time.Duration((float64(ts.Second) + float64(ts.Fraction)/fractionDivisor) * float64(time.Second))
}

// IsAfter returns true if this timestamp is later than the given one.
func (ts Timestamp) IsAfter(other Timestamp) bool {
	return (ts.Second > other.Second) || ((ts.Second == other.Second) && (ts.Fraction > other.Fraction))
}