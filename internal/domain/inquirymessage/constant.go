package inquirymessage

const (
	// minBodyLength は、本文の最小文字数（rune 数）です。空の投稿を認めません。
	minBodyLength = 1

	// maxBodyLength は、本文の最大文字数（rune 数）です。
	// UTF-8 で最大 3 バイト × この値としても、Realtime Delivery の payload 上限 64 KiB を
	// event の封筒と送り手を足しても超えません。
	maxBodyLength = 4000
)
