package shapes

type HideNames map[string]bool

func (n HideNames) IsIn(name string) bool {
	_, ok := n[name]
	return ok
}

func (n HideNames) Add(name string) {
	n[name] = true
}

func (n HideNames) Delete(name string) {
	if n.IsIn(name) {
		delete(n, name)
	}
}

func (n HideNames) DeleteAll() {
	for s := range n {
		n.Delete(s)
	}
}
