package strmatcher

import (
	"regexp"
	"strings"
)

type MapMatcherGroup struct {
	otherMatchers []matcherEntry
	count         uint32
	keywordsList  []string
	ruleMap       map[string]bool
}

// pattern is guaranteed to be lowercased
func (g *MapMatcherGroup) AddFullOrDomainPattern(pattern string, t Type) {
	switch t {
	case Domain:
		g.ruleMap["."+pattern] = true
		fallthrough
	case Full:
		g.ruleMap[pattern] = true
	default:
	}
}

func NewMapMatcherGroup() *MapMatcherGroup {
	return &MapMatcherGroup{
		otherMatchers: nil,
		count:         0,
		keywordsList:  nil,
		ruleMap:       make(map[string]bool),
	}
}

// AddPattern adds a pattern to MphMatcherGroup
func (g *MapMatcherGroup) AddPattern(pattern string, t Type) (uint32, error) {
	switch t {
	case Substr:
		g.keywordsList = append(g.keywordsList, pattern)
	case Full, Domain:
		pattern = strings.ToLower(pattern)
		g.AddFullOrDomainPattern(pattern, t)
	case Regex:
		r, err := regexp.Compile(pattern)
		if err != nil {
			return 0, err
		}
		g.otherMatchers = append(g.otherMatchers, matcherEntry{
			m:  &regexMatcher{pattern: r},
			id: g.count,
		})
	default:
		panic("Unknown type")
	}
	return g.count, nil
}

// Build builds a minimal perfect hash table and ac automaton from insert rules
func (g *MapMatcherGroup) Build() {

}

// Lookup searches for s in t and returns its index and whether it was found.
func (g *MapMatcherGroup) Lookup(pattern string) bool {
	if _, ok := g.ruleMap[pattern]; ok {
		return true
	}
	return false
}

// Match implements IndexMatcher.Match.
func (g *MapMatcherGroup) Match(pattern string) []uint32 {
	result := []uint32{}
	for i := len(pattern) - 1; i >= 0; i-- {
		if pattern[i] == '.' {
			if g.Lookup(pattern[i:]) {
				result = append(result, 1)
				return result
			}
		}
	}
	if g.Lookup(pattern) {
		result = append(result, 1)
		return result
	}
	if len(g.keywordsList) > 0 {
		for _, keyword := range g.keywordsList {
			if strings.Contains(pattern, keyword) {
				result = append(result, 1)
				return result
			}
		}
	}
	for _, e := range g.otherMatchers {
		if e.m.Match(pattern) {
			result = append(result, e.id)
			return result
		}
	}
	return nil
}
