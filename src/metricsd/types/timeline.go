package types

import (
	"fmt"
	"sync"
	"time"
)

// A Timeline is used to store events in a list of slices, divided by the
// time they have been taken at.
type Timeline struct {
	Interval int64
	Slices   map[int64]*Slice
	mutex    *sync.Mutex
}

// NewTimeline returns a new timeline Timeline with the given slice interval.
func NewTimeline(sliceInterval int) *Timeline {
	return &Timeline{
		Slices:   make(map[int64]*Slice),
		Interval: int64(sliceInterval),
		mutex:    &sync.Mutex{},
	}
}

// Add appends the given event to the current slice.
func (timeline *Timeline) Add(event *Event) {
	timeline.getCurrentSlice().Add(event)
}

func (timeline *Timeline) ExtractClosedSlices(force bool) (closedSlices []*Slice) {
	var current int64
	if force {
		current = -1
	} else {
		current = timeline.getCurrentSliceNumber()
	}

	// Calculate total number of closed timeline (to avoid vector reallocs)
	totalClosedSlices := 0
	timeline.eachClosedSlice(current, func(number int64, slice *Slice) {
		totalClosedSlices += 1
	})

	// Create an array to store timeline
	closedSlices = make([]*Slice, 0, totalClosedSlices)
	timeline.eachClosedSlice(current, func(number int64, slice *Slice) {
		closedSlices = append(closedSlices, slice)
		timeline.mutex.Lock()
		timeline.Slices[number] = nil, false
		timeline.mutex.Unlock()
	})
	SortSlices(closedSlices)
	return
}

// ExtractClosedSampleSets finds closed timeline, and stores all sample sets from them
// in an array. Processed timeline will be removed from the list of active timeline.
func (timeline *Timeline) ExtractClosedSampleSets(force bool) (closedSampleSets []*SampleSet) {
	var current int64
	if force {
		current = -1
	} else {
		current = timeline.getCurrentSliceNumber()
	}

	// Calculate total number of closed sample sets (to avoid vector reallocs)
	totalSampleSets := 0
	timeline.eachClosedSlice(current, func(number int64, slice *Slice) {
		totalSampleSets += len(slice.Sets)
	})

	// Create an array to store sample sets
	closedSampleSets = make([]*SampleSet, 0, totalSampleSets)
	timeline.eachClosedSlice(current, func(number int64, slice *Slice) {
		for _, set := range slice.Sets {
			closedSampleSets = append(closedSampleSets, set)
		}
		timeline.mutex.Lock()
		timeline.Slices[number] = nil, false
		timeline.mutex.Unlock()
	})
	SortSampleSets(closedSampleSets)
	return
}

func (timeline *Timeline) String() string {
	return fmt.Sprintf(
		"Timeline[interval=%d, size=%d]",
		timeline.Interval,
		len(timeline.Slices),
	)
}

// getCurrentSlice creates (if necessary) and returns the current slice
// (see getCurrentSliceNumber for details).
func (timeline *Timeline) getCurrentSlice() *Slice {
	number := timeline.getCurrentSliceNumber()
	if _, found := timeline.Slices[number]; !found {
		timeline.mutex.Lock()
		timeline.Slices[number] = NewSlice(number * timeline.Interval)
		timeline.mutex.Unlock()
	}
	return timeline.Slices[number]
}

// getCurrentSliceNumber returns current slice number (time since epoc in
// seconds, rounded to the slices interval).
func (timeline *Timeline) getCurrentSliceNumber() int64 {
	return time.Seconds() / timeline.Interval
}

// eachClosedSlice calls function f for each slice with the slice number less
// then current, in no particular order. If current is negative, it calls
// function f for all slices.
func (timeline *Timeline) eachClosedSlice(current int64, f func(number int64, slice *Slice)) {
	for number, slice := range timeline.Slices {
		if number < current || current < 0 {
			f(number, slice)
		}
	}
}
