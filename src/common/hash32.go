package common

import "hash/fnv"

func Hash32(data []byte) uint32 {
	h := fnv.New32a()

	h.Write(data)

	return h.Sum32()
}
