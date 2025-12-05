package scheduler

type jobHeap []*CheckJob

func (h jobHeap) Len() int           { return len(h) }
func (h jobHeap) Less(i, j int) bool { return h[i].nextRun.Before(h[j].nextRun) }
func (h jobHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *jobHeap) Push(x any) {
	n := len(*h)
	item := x.(*CheckJob)
	item.index = n
	*h = append(*h, item)
}
func (h *jobHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	item.index = -1
	*h = old[0 : n-1]
	return item
}
