package ut

import (
	"github.com/OneOfOne/xxhash"
	"github.com/itchyny/base58-go"
	"github.com/jxskiss/base62"
	"io"
	"strconv"
	"strings"
)

func StrToInt(s string) int64 {
	h := xxhash.New32()
	r := strings.NewReader(s)
	_, _ = io.Copy(h, r)
	keyInt := h.Sum32()
	return int64(keyInt)
}

func StrToInt64(s string) int64 {
	h := xxhash.New64()
	r := strings.NewReader(s)
	_, _ = io.Copy(h, r)
	keyInt := h.Sum64()
	return int64(keyInt)
}

func StrToB62(s string) string {
	d := strconv.FormatUint(uint64(StrToInt(s)), 10)
	return base62.EncodeToString([]byte(d))
}

func StrToB58(s string) string {
	hex := StrToInt(s)
	encoding := base58.FlickrEncoding
	strByte := []byte(strconv.FormatInt(hex, 10))
	encoded, _ := encoding.Encode(strByte)
	return string(encoded)
}
