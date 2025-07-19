package testctxlint

import (
	"go/ast"
	"go/token"
	"sort"
)

type scope struct {
	// Scope-defining node
	ast.Node

	// The function that declares this scope
	funcType *ast.FuncType

	// Parent scope or nil
	parent *scope
}

func (s *scope) isAncestorOf(sub *scope) bool {
	for p := sub.parent; p != nil; p = p.parent {
		if p == s {
			return true
		}
	}

	return false
}

func (s *scope) findNearestBenchmarkOrTestParamWithInfo() *testingParam {
	for current := s; current != nil; current = current.parent {
		if tp := benchmarkOrTestParamWithInfo(current.funcType); tp != nil {
			return tp
		}
	}

	return nil
}

// scopeCollection holds scopes sorted by position for efficient lookup
type scopeCollection struct {
	scopes []*scope
	sorted bool
}

func (sc *scopeCollection) add(s *scope) {
	sc.scopes = append(sc.scopes, s)
	sc.sorted = false
}

func (sc *scopeCollection) ensureSorted() {
	if !sc.sorted {
		sort.Slice(sc.scopes, func(i, j int) bool {
			return sc.scopes[i].Pos() < sc.scopes[j].Pos()
		})
		sc.sorted = true
	}
}

func (sc *scopeCollection) findScope(pos token.Pos) *scope {
	if len(sc.scopes) == 0 {
		return nil
	}

	sc.ensureSorted()

	var closestScope *scope

	// Use binary search to find candidate scopes efficiently
	// Find the rightmost scope that starts at or before pos
	left, right := 0, len(sc.scopes)
	for left < right {
		mid := (left + right) / 2
		if sc.scopes[mid].Pos() <= pos {
			left = mid + 1
		} else {
			right = mid
		}
	}

	// Check scopes starting from the rightmost candidate and work backwards
	// This is efficient because scopes are sorted by start position
	for i := left - 1; i >= 0; i-- {
		s := sc.scopes[i]

		// Skip scopes that don't contain this position
		if s.Pos() > pos || pos > s.End() {
			continue
		}

		// Skip scopes that are less specific than our current best
		if closestScope != nil && s.isAncestorOf(closestScope) {
			continue
		}

		closestScope = s

		// Since scopes are sorted by start position, we can break early
		// if we find a scope that contains the position, as any earlier
		// scopes will be less specific or won't contain the position
		if closestScope != nil {
			break
		}
	}

	return closestScope
}
