package structs

const base = 0x0110000100000000

func IdToNpid(id int) uint64 {
	return base + uint64(id)
}

func NpidToId(npid uint64) int {
	return int(npid - base)
}
