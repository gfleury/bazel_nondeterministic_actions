package execlog

import (
	"bufio"
	"container/heap"
	"io"

	pb "tools/execlog/proto"
	"google.golang.org/protobuf/encoding/protodelim"
)

// ReadSpawnExec reads a single varint-delimited SpawnExec message from reader.
// Returns (nil, nil) at EOF.
func ReadSpawnExec(reader *bufio.Reader) (*pb.SpawnExec, error) {
	exec := &pb.SpawnExec{}
	err := protodelim.UnmarshalFrom(reader, exec)
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return exec, nil
}

// Parser reads SpawnExec messages one at a time. Returns (nil, nil) when done.
type Parser interface {
	Next() (*pb.SpawnExec, error)
}

// FilteringParser reads delimited SpawnExec messages, optionally filtering by runner.
type FilteringParser struct {
	reader           *bufio.Reader
	restrictToRunner string
}

// NewFilteringParser creates a parser that reads varint-delimited SpawnExec messages
// from r, optionally filtering to only those with the given runner.
func NewFilteringParser(r io.Reader, restrictToRunner string) *FilteringParser {
	return &FilteringParser{
		reader:           bufio.NewReader(r),
		restrictToRunner: restrictToRunner,
	}
}

func (p *FilteringParser) Next() (*pb.SpawnExec, error) {
	for {
		exec, err := ReadSpawnExec(p.reader)
		if err != nil || exec == nil {
			return nil, err
		}
		if p.restrictToRunner == "" || exec.Runner == p.restrictToRunner {
			return exec, nil
		}
	}
}

// Golden tracks the ordering of SpawnExec records from the first file
// so the second file can be reordered to match.
type Golden struct {
	positions map[string]int
	index     int
}

func NewGolden() *Golden {
	return &Golden{positions: make(map[string]int)}
}

// AddSpawnExec records the position of an exec by its first listed output.
func (g *Golden) AddSpawnExec(exec *pb.SpawnExec) {
	key := getFirstOutput(exec)
	if key != "" {
		g.positions[key] = g.index
		g.index++
	}
}

// PositionFor returns the golden position for an exec, or -1 if not found.
func (g *Golden) PositionFor(exec *pb.SpawnExec) int {
	key := getFirstOutput(exec)
	if key != "" {
		if pos, ok := g.positions[key]; ok {
			return pos
		}
	}
	return -1
}

func getFirstOutput(exec *pb.SpawnExec) string {
	if len(exec.ListedOutputs) > 0 {
		return exec.ListedOutputs[0]
	}
	return ""
}

// GetFirstOutput returns the first listed output of a SpawnExec, or "" if none.
func GetFirstOutput(exec *pb.SpawnExec) string {
	return getFirstOutput(exec)
}

// Priority queue for reordering SpawnExec records by golden position.

type element struct {
	position int
	exec     *pb.SpawnExec
}

type priorityQueue []*element

func (pq priorityQueue) Len() int            { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool  { return pq[i].position < pq[j].position }
func (pq priorityQueue) Swap(i, j int)       { pq[i], pq[j] = pq[j], pq[i] }
func (pq *priorityQueue) Push(x interface{}) { *pq = append(*pq, x.(*element)) }
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}

// ReorderingParser emits SpawnExec records reordered to match the golden
// ordering, followed by any records not found in the golden set.
type ReorderingParser struct {
	sameActions   priorityQueue
	uniqueActions []*pb.SpawnExec
	uniqueIdx     int
}

// NewReorderingParser reads all records from input, partitions them into
// those matching the golden ordering and those unique to this file,
// then serves them in golden order followed by unique actions.
func NewReorderingParser(golden *Golden, input Parser) (*ReorderingParser, error) {
	rp := &ReorderingParser{}
	pq := &priorityQueue{}
	heap.Init(pq)

	for {
		exec, err := input.Next()
		if err != nil {
			return nil, err
		}
		if exec == nil {
			break
		}
		position := golden.PositionFor(exec)
		if position >= 0 {
			heap.Push(pq, &element{position: position, exec: exec})
		} else {
			rp.uniqueActions = append(rp.uniqueActions, exec)
		}
	}
	rp.sameActions = *pq
	return rp, nil
}

func (rp *ReorderingParser) Next() (*pb.SpawnExec, error) {
	if rp.sameActions.Len() > 0 {
		elem := heap.Pop(&rp.sameActions).(*element)
		return elem.exec, nil
	}
	if rp.uniqueIdx < len(rp.uniqueActions) {
		exec := rp.uniqueActions[rp.uniqueIdx]
		rp.uniqueIdx++
		return exec, nil
	}
	return nil, nil
}
