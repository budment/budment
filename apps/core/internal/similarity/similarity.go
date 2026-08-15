package similarity

import (
	"strings"

	"github.com/xrash/smetrics"
)

// Matcher establishes the contract for semantic string comparison.
type Matcher interface {
	Calculate(a, b string) float64
}

// JaroWinkler optimally matches API fields by rewarding common prefixes.
// It returns a score between 0.0 and 1.0
type JaroWinkler struct {
	BoostThreshold float64
	PrefixSize     int
}

func NewJaroWinkler(boostThreshold float64, prefixSize int) *JaroWinkler {
	return &JaroWinkler{
		BoostThreshold: boostThreshold,
		PrefixSize:     prefixSize,
	}
}

func (j *JaroWinkler) Calculate(a, b string) float64 {
	// Boost threshold 0.7, Prefix size 4 - standard for short variable names
	return smetrics.JaroWinkler(strings.ToLower(a), strings.ToLower(b), j.BoostThreshold, j.PrefixSize)
}

// TypeCompatibility dictates whether an Identity mapping is logically sound.
type TypeCompatibility interface {
	Compatible(consumerTypes, producerTypes []string) bool
}

type DefaultTypeCompatibility struct{}

func NewDefaultTypeCompatibility() *DefaultTypeCompatibility {
	return &DefaultTypeCompatibility{}
}

func (t *DefaultTypeCompatibility) Compatible(consumerTypes, producerTypes []string) bool {
	if len(consumerTypes) == 0 || len(producerTypes) == 0 {
		return true
	}
	pMap := make(map[string]bool, len(producerTypes))
	for _, pt := range producerTypes {
		pMap[pt] = true
	}
	for _, ct := range consumerTypes {
		if pMap[ct] {
			return true
		}
		if (ct == "string" && pMap["integer"]) || (ct == "integer" && pMap["string"]) {
			return true
		}
	}
	return false
}
