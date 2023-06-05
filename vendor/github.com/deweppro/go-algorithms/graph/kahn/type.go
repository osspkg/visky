// see: https://en.wikipedia.org/wiki/Topological_sorting

package kahn

import "errors"

var (
	ErrBuildKahn = errors.New("can't do topographical sorting")
)

type Graph struct {
	graph  map[string]map[string]int
	tmp    map[string]bool
	result []string
}

func New() *Graph {
	return &Graph{
		graph:  make(map[string]map[string]int),
		tmp:    make(map[string]bool),
		result: make([]string, 0),
	}
}

// Add - Adding a graph edge
func (k *Graph) Add(from, to string) error {
	if _, ok := k.graph[from]; !ok {
		k.graph[from] = make(map[string]int)
	}
	k.graph[from][to]++
	return nil
}

// To update the temporary map
func (k *Graph) updateTemp() int {
	for i, sub := range k.graph {
		for j := range sub {
			k.tmp[j] = true
		}
		k.tmp[i] = true
	}
	return len(k.tmp)
}

// Build - Perform sorting
func (k *Graph) Build() error {
	k.result = k.result[:0]
	length := k.updateTemp()
	for len(k.result) < length {
		found := ""
		for item := range k.tmp {
			if k.find(item) {
				found = item
				break
			}
		}
		if len(found) > 0 {
			k.result = append(k.result, found)
			delete(k.tmp, found)
		} else {
			return ErrBuildKahn
		}
	}
	return nil
}

// Finding the next edge
func (k *Graph) find(item string) bool {
	for i, j := range k.graph {
		if _, jok := j[item]; jok {
			if _, iok := k.tmp[i]; iok {
				return false
			}
		}
	}
	return true
}

// Result - Getting a sorted slice
func (k *Graph) Result() []string {
	return k.result
}
