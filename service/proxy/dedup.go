package proxy

// dedup is a ring buffer containing the last 128 ids used to check for
// duplicates.
type dedup struct {
	ids [128]uint16
	pos int
}

// IsDup returns true if id is present in the ring buffer. If not found, the id
// is added to the ring. Call to this function is not thread safe.
func (d *dedup) IsDup(id uint16) bool {
	for _, id2 := range d.ids {
		if id == id2 {
			return true
		}
	}
	d.ids[d.pos] = id
	d.pos++
	d.pos &= 127 // power of 2 modulo
	return false
}
